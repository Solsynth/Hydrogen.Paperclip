package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	pkg "git.solsynth.dev/hypernet/paperclip/pkg/internal"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/fs"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/gap"
	"github.com/fatih/color"

	"git.solsynth.dev/hypernet/paperclip/pkg/internal/cache"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/grpc"

	"git.solsynth.dev/hypernet/paperclip/pkg/internal/server"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/services"
	"github.com/robfig/cron/v3"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func main() {
	// Booting screen
	fmt.Println(color.YellowString(" ____                           _ _\n|  _ \\ __ _ _ __   ___ _ __ ___| (_)_ __\n| |_) / _` | '_ \\ / _ \\ '__/ __| | | '_ \\\n|  __/ (_| | |_) |  __/ | | (__| | | |_) |\n|_|   \\__,_| .__/ \\___|_|  \\___|_|_| .__/\n           |_|                     |_|"))
	fmt.Printf("%s v%s\n", color.New(color.FgHiYellow).Add(color.Bold).Sprintf("Hypernet.Paperclip"), pkg.AppVersion)
	fmt.Printf("The upload service in Hypernet\n")
	color.HiBlack("=====================================================\n")

	// Configure settings
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.SetConfigName("settings")
	viper.SetConfigType("toml")

	// Load settings
	if err := viper.ReadInConfig(); err != nil {
		log.Panic().Err(err).Msg("An error occurred when loading settings.")
	}

	// Connect to nexus
	if err := gap.InitializeToNexus(); err != nil {
		log.Error().Err(err).Msg("An error occurred when registering service to nexus...")
	}

	// Load keypair
	if reader, err := sec.NewInternalTokenReader(viper.GetString("security.internal_public_key")); err != nil {
		log.Error().Err(err).Msg("An error occurred when reading internal public key for jwt. Authentication related features will be disabled.")
	} else {
		server.IReader = reader
		log.Info().Msg("Internal jwt public key loaded.")
	}

	// Connect to database
	if err := database.NewGorm(); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when connect to database.")
	} else if err := database.RunMigration(database.C); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when running database auto migration.")
	}

	// Initialize cache
	if err := cache.NewStore(); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when initializing cache.")
	}

	// Set up some workers
	for idx := 0; idx < viper.GetInt("workers.files_analyze"); idx++ {
		go services.StartConsumeAnalyzeTask()
	}

	// Configure timed tasks
	quartz := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(&log.Logger)))
	quartz.AddFunc("@every 60m", services.DoAutoDatabaseCleanup)
	quartz.AddFunc("@every 60m", fs.RunMarkLifecycleDeletionTask)
	quartz.AddFunc("@every 60m", fs.RunMarkMultipartDeletionTask)
	quartz.AddFunc("@midnight", fs.RunScheduleDeletionTask)
	quartz.Start()

	// Server
	go server.NewServer().Listen()

	// Grpc Server
	go grpc.NewGrpc().Listen()

	// Post-boot actions
	services.BuildDestinationMapping()
	services.ScanUnanalyzedFileFromDatabase()
	fs.RunMarkLifecycleDeletionTask()

	// Messages
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	quartz.Stop()
}
