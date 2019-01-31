package main

import (
	"io"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/elastos/Elastos.ELA/distributor"

	"github.com/elastos/Elastos.ELA.Utility/elalog"
	"github.com/elastos/Elastos.ELA.Utility/signal"
)

const (
	// logDir is the directory to put distributor log files.
	logDir = "logs"

	// maxLogFileSize is the maximum size for a single log file.
	maxLogFileSize = 5 * elalog.MBSize

	// maxLogFolderSize is the maximum folder size for all log files.
	maxLogFolderSize = 100 * elalog.MBSize
)

var (
	// Build version generated when build program.
	Version string

	// The go source code version at build.
	GoVersion string
)

// loads config file and start the distributor program.  One computer should
// only run one distributor instance.
func main() {
	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Block and transaction processing can cause bursty allocations.  This
	// limits the garbage collector from excessively overallocating during
	// bursts.  This value was arrived at with the help of profiling live
	// usage.
	debug.SetGCPercent(10)

	// Set logger for distributor package.
	writer := elalog.NewFileWriter(logDir, maxLogFileSize, maxLogFolderSize)
	backend := elalog.NewBackend(io.MultiWriter(os.Stdout, writer))
	log := backend.Logger("", elalog.LevelInfo)
	distributor.UseLogger(log)

	log.Infof("Node version: %s", Version)
	log.Info(GoVersion)

	interrupt := signal.NewInterrupt()

	mapping, err := loadConfig()
	if err != nil {
		logAndExit(log, err)
	}

	d := distributor.New()
	for port, addr := range mapping {
		if err := d.Mapping(port, addr); err != nil {
			logAndExit(log, err)
		}
	}

	d.Start()
	<-interrupt.C
	d.Stop()
}

// logAndExit output error info into log file and exit program.
func logAndExit(log elalog.Logger, err error) {
	log.Error(err)
	os.Exit(-1)
}
