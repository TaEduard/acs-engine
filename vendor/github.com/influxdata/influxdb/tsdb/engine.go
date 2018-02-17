package tsdb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/influxdata/influxdb/models"
	"github.com/influxdata/influxdb/pkg/estimator"
	"github.com/influxdata/influxdb/pkg/limiter"
	"github.com/influxdata/influxdb/query"
	"github.com/influxdata/influxql"
	"go.uber.org/zap"
)

var (
	// ErrFormatNotFound is returned when no format can be determined from a path.
	ErrFormatNotFound = errors.New("format not found")

	// ErrUnknownEngineFormat is returned when the engine format is
	// unknown. ErrUnknownEngineFormat is currently returned if a format
	// other than tsm1 is encountered.
	ErrUnknownEngineFormat = errors.New("unknown engine format")
)

// Engine represents a swappable storage engine for the shard.
type Engine interface {
	Open() error
	Close() error
	SetEnabled(enabled bool)
	SetCompactionsEnabled(enabled bool)
	ScheduleFullCompaction() error

	WithLogger(*zap.Logger)

	LoadMetadataIndex(shardID uint64, index Index) error

	CreateSnapshot() (string, error)
	Backup(w io.Writer, basePath string, since time.Time) error
	Export(w io.Writer, basePath string, start time.Time, end time.Time) error
	Restore(r io.Reader, basePath string) error
	Import(r io.Reader, basePath string) error
	Digest() (io.ReadCloser, int64, error)

	CreateIterator(ctx context.Context, measurement string, opt query.IteratorOptions) (query.Iterator, error)
	CreateCursor(ctx context.Context, r *CursorRequest) (Cursor, error)
	IteratorCost(measurement string, opt query.IteratorOptions) (query.IteratorCost, error)
	WritePoints(points []models.Point) error

	CreateSeriesIfNotExists(key, name []byte, tags models.Tags) error
	CreateSeriesListIfNotExists(keys, names [][]byte, tags []models.Tags) error
	DeleteSeriesRange(itr SeriesIterator, min, max int64, removeIndex bool) error

	MeasurementsSketches() (estimator.Sketch, estimator.Sketch, error)
	SeriesN() int64

	MeasurementExists(name []byte) (bool, error)

	MeasurementNamesByRegex(re *regexp.Regexp) ([][]byte, error)
	MeasurementFieldSet() *MeasurementFieldSet
	MeasurementFields(measurement []byte) *MeasurementFields
	ForEachMeasurementName(fn func(name []byte) error) error
	DeleteMeasurement(name []byte) error

	HasTagKey(name, key []byte) (bool, error)
	MeasurementTagKeysByExpr(name []byte, expr influxql.Expr) (map[string]struct{}, error)
	TagKeyCardinality(name, key []byte) int

	// Statistics will return statistics relevant to this engine.
	Statistics(tags map[string]string) []models.Statistic
	LastModified() time.Time
	DiskSize() int64
	IsIdle() bool
	Free() error

	io.WriterTo
}

// EngineFormat represents the format for an engine.
type EngineFormat int

const (
	// TSM1Format is the format used by the tsm1 engine.
	TSM1Format EngineFormat = 2
)

// NewEngineFunc creates a new engine.
type NewEngineFunc func(id uint64, i Index, database, path string, walPath string, sfile *SeriesFile, options EngineOptions) Engine

// newEngineFuncs is a lookup of engine constructors by name.
var newEngineFuncs = make(map[string]NewEngineFunc)

// RegisterEngine registers a storage engine initializer by name.
func RegisterEngine(name string, fn NewEngineFunc) {
	if _, ok := newEngineFuncs[name]; ok {
		panic("engine already registered: " + name)
	}
	newEngineFuncs[name] = fn
}

// RegisteredEngines returns the slice of currently registered engines.
func RegisteredEngines() []string {
	a := make([]string, 0, len(newEngineFuncs))
	for k := range newEngineFuncs {
		a = append(a, k)
	}
	sort.Strings(a)
	return a
}

// NewEngine returns an instance of an engine based on its format.
// If the path does not exist then the DefaultFormat is used.
func NewEngine(id uint64, i Index, database, path string, walPath string, sfile *SeriesFile, options EngineOptions) (Engine, error) {
	// Create a new engine
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return newEngineFuncs[options.EngineVersion](id, i, database, path, walPath, sfile, options), nil
	}

	// If it's a dir then it's a tsm1 engine
	format := DefaultEngine
	if fi, err := os.Stat(path); err != nil {
		return nil, err
	} else if !fi.Mode().IsDir() {
		return nil, ErrUnknownEngineFormat
	} else {
		format = "tsm1"
	}

	// Lookup engine by format.
	fn := newEngineFuncs[format]
	if fn == nil {
		return nil, fmt.Errorf("invalid engine format: %q", format)
	}

	return fn(id, i, database, path, walPath, sfile, options), nil
}

// EngineOptions represents the options used to initialize the engine.
type EngineOptions struct {
	EngineVersion string
	IndexVersion  string
	ShardID       uint64
	InmemIndex    interface{} // shared in-memory index

	CompactionLimiter           limiter.Fixed
	CompactionThroughputLimiter limiter.Rate

	Config Config
}

// NewEngineOptions returns the default options.
func NewEngineOptions() EngineOptions {
	return EngineOptions{
		EngineVersion: DefaultEngine,
		IndexVersion:  DefaultIndex,
		Config:        NewConfig(),
	}
}

// NewInmemIndex returns a new "inmem" index type.
var NewInmemIndex func(name string, sfile *SeriesFile) (interface{}, error)