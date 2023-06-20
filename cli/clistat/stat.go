package clistat

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/elastic/go-sysinfo"
	"github.com/spf13/afero"
	"golang.org/x/xerrors"
	"tailscale.com/types/ptr"

	sysinfotypes "github.com/elastic/go-sysinfo/types"
)

// Result is a generic result type for a statistic.
// Total is the total amount of the resource available.
// It is nil if the resource is not a finite quantity.
// Unit is the unit of the resource.
// Used is the amount of the resource used.
type Result struct {
	Total *float64 `json:"total"`
	Unit  string   `json:"unit"`
	Used  float64  `json:"used"`
}

// String returns a human-readable representation of the result.
func (r *Result) String() string {
	if r == nil {
		return "-"
	}

	var usedDisplay, totalDisplay string
	var usedScaled, totalScaled float64
	var usedPrefix, totalPrefix string
	usedScaled, usedPrefix = humanize.ComputeSI(r.Used)
	usedDisplay = humanizeFloat(usedScaled)
	if r.Total != (*float64)(nil) {
		totalScaled, totalPrefix = humanize.ComputeSI(*r.Total)
		totalDisplay = humanizeFloat(totalScaled)
	}

	var sb strings.Builder
	_, _ = sb.WriteString(usedDisplay)

	// If the unit prefixes of the used and total values are different,
	// display the used value's prefix to avoid confusion.
	if usedPrefix != totalPrefix || totalDisplay == "" {
		_, _ = sb.WriteString(" ")
		_, _ = sb.WriteString(usedPrefix)
		_, _ = sb.WriteString(r.Unit)
	}

	if totalDisplay != "" {
		_, _ = sb.WriteString("/")
		_, _ = sb.WriteString(totalDisplay)
		_, _ = sb.WriteString(" ")
		_, _ = sb.WriteString(totalPrefix)
		_, _ = sb.WriteString(r.Unit)
	}

	if r.Total != nil && *r.Total != 0.0 {
		_, _ = sb.WriteString(" (")
		_, _ = sb.WriteString(fmt.Sprintf("%.0f", r.Used/(*r.Total)*100.0))
		_, _ = sb.WriteString("%)")
	}

	return strings.TrimSpace(sb.String())
}

func humanizeFloat(f float64) string {
	// humanize.FtoaWithDigits does not round correctly.
	prec := precision(f)
	rat := math.Pow(10, float64(prec))
	rounded := math.Round(f*rat) / rat
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

// limit precision to 3 digits at most to preserve space
func precision(f float64) int {
	fabs := math.Abs(f)
	if fabs == 0.0 {
		return 0
	}
	if fabs < 1.0 {
		return 3
	}
	if fabs < 10.0 {
		return 2
	}
	if fabs < 100.0 {
		return 1
	}
	return 0
}

// Statter is a system statistics collector.
// It is a thin wrapper around the elastic/go-sysinfo library.
type Statter struct {
	hi             sysinfotypes.Host
	fs             afero.Fs
	sampleInterval time.Duration
	nproc          int
	wait           func(time.Duration)
}

type Option func(*Statter)

// WithSampleInterval sets the sample interval for the statter.
func WithSampleInterval(d time.Duration) Option {
	return func(s *Statter) {
		s.sampleInterval = d
	}
}

// WithFS sets the fs for the statter.
func WithFS(fs afero.Fs) Option {
	return func(s *Statter) {
		s.fs = fs
	}
}

func New(opts ...Option) (*Statter, error) {
	hi, err := sysinfo.Host()
	if err != nil {
		return nil, xerrors.Errorf("get host info: %w", err)
	}
	s := &Statter{
		hi:             hi,
		fs:             afero.NewReadOnlyFs(afero.NewOsFs()),
		sampleInterval: 100 * time.Millisecond,
		nproc:          runtime.NumCPU(),
		wait: func(d time.Duration) {
			<-time.After(d)
		},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// HostCPU returns the CPU usage of the host. This is calculated by
// taking two samples of CPU usage and calculating the difference.
// Total will always be equal to the number of cores.
// Used will be an estimate of the number of cores used during the sample interval.
// This is calculated by taking the difference between the total and idle HostCPU time
// and scaling it by the number of cores.
// Units are in "cores".
func (s *Statter) HostCPU() (*Result, error) {
	r := &Result{
		Unit:  "cores",
		Total: ptr.To(float64(s.nproc)),
	}
	c1, err := s.hi.CPUTime()
	if err != nil {
		return nil, xerrors.Errorf("get first cpu sample: %w", err)
	}
	s.wait(s.sampleInterval)
	c2, err := s.hi.CPUTime()
	if err != nil {
		return nil, xerrors.Errorf("get second cpu sample: %w", err)
	}
	total := c2.Total() - c1.Total()
	if total == 0 {
		return r, nil // no change
	}
	idle := c2.Idle - c1.Idle
	used := total - idle
	scaleFactor := float64(s.nproc) / total.Seconds()
	r.Used = used.Seconds() * scaleFactor
	return r, nil
}

// HostMemory returns the memory usage of the host, in gigabytes.
func (s *Statter) HostMemory() (*Result, error) {
	r := &Result{
		Unit: "B",
	}
	hm, err := s.hi.Memory()
	if err != nil {
		return nil, xerrors.Errorf("get memory info: %w", err)
	}
	r.Total = ptr.To(float64(hm.Total))
	// On Linux, hm.Used equates to MemTotal - MemFree in /proc/stat.
	// This includes buffers and cache.
	// So use MemAvailable instead, which only equates to physical memory.
	// On Windows, this is also calculated as Total - Available.
	r.Used = float64(hm.Total - hm.Available)
	return r, nil
}
