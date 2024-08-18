package main

import (
	"os"
	"os/signal"
	"syscall"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/grpc"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/server"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/services"
	"github.com/robfig/cron/v3"

	pkg "git.solsynth.dev/hydrogen/paperclip/pkg/internal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func main() {
	// Configure settings
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.SetConfigName("settings")
	viper.SetConfigType("toml")

	// Load settings
	if err := viper.ReadInConfig(); err != nil {
		log.Panic().Err(err).Msg("An error occurred when loading settings.")
	}

	// Connect to database
	if err := database.NewSource(); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when connect to database.")
	} else if err := database.RunMigration(database.C); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when running database auto migration.")
	}

	// Connect other services
	if err := gap.RegisterService(); err != nil {
		log.Error().Err(err).Msg("An error occurred when registering service to dealer...")
	}

	// Set up some workers
	for idx := 0; idx < viper.GetInt("workers.files_deletion"); idx++ {
		go services.StartConsumeDeletionTask()
	}
	for idx := 0; idx < viper.GetInt("workers.files_analyze"); idx++ {
		go services.StartConsumeAnalyzeTask()
	}

	// Configure timed tasks
	quartz := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(&log.Logger)))
	quartz.AddFunc("@every 60m", services.DoAutoDatabaseCleanup)
	quartz.AddFunc("@every 60m", services.RunMarkDeletionTask)
	quartz.AddFunc("@midnight", services.RunScheduleDeletionTask)
	quartz.Start()

	// Server
	server.NewServer()
	go server.Listen()

	// Grpc Server
	grpc.NewGRPC()
	go grpc.ListenGRPC()

	// Messages
	log.Info().Msgf("Paperclip v%s is started...", pkg.AppVersion)

	services.ScanUnanalyzedFileFromDatabase()
	services.RunMarkDeletionTask()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msgf("Paperclip v%s is quitting...", pkg.AppVersion)

	quartz.Stop()
}
