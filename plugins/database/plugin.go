// Package database is a plugin that manages the badger database (e.g. garbage collection).
package database

import (
	"errors"
	"sync"
	"time"

	"github.com/iotaledger/wasp/packages/parameters"

	"github.com/iotaledger/goshimmer/packages/database"
	"github.com/iotaledger/hive.go/daemon"
	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/node"
	"github.com/iotaledger/hive.go/timeutil"
)

const pluginName = "Database"

var (
	log *logger.Logger

	db        database.DB
	store     kvstore.KVStore
	storeOnce sync.Once
)

// Init is an entry point for the plugin.
func Init() *node.Plugin {
	return node.NewPlugin(pluginName, node.Enabled, configure, run)
}

func configure(_ *node.Plugin) {
	// assure that the store is initialized
	_ = storeInstance()

	err := checkDatabaseVersion()
	if errors.Is(err, ErrDBVersionIncompatible) {
		log.Panicf("The database scheme was updated. Please delete the database folder.\n%s", err)
	}
	if err != nil {
		log.Panicf("Failed to check database version: %s", err)
	}

	// we open the database in the configure, so we must also make sure it's closed here
	err = daemon.BackgroundWorker(pluginName, closeDB, parameters.PriorityDatabase)
	if err != nil {
		log.Panicf("Failed to start as daemon: %s", err)
	}
}

func run(_ *node.Plugin) {
	if err := daemon.BackgroundWorker(pluginName+"[GC]", runGC, parameters.PriorityBadgerGarbageCollection); err != nil {
		log.Errorf("Failed to start as daemon: %s", err)
	}
}

func closeDB(shutdownSignal <-chan struct{}) {
	<-shutdownSignal
	log.Infof("Syncing database to disk...")
	if err := db.Close(); err != nil {
		log.Errorf("Failed to flush the database: %s", err)
	}
	log.Infof("Syncing database to disk... done")
}

func runGC(shutdownSignal <-chan struct{}) {
	if !db.RequiresGC() {
		return
	}
	// run the garbage collection with the given interval
	timeutil.NewTicker(func() {
		if err := db.GC(); err != nil {
			log.Warnf("Garbage collection failed: %s", err)
		}
	}, 5*time.Minute, shutdownSignal)
}
