package main

import (
	"embed"
	"log"

	"byteflow-studio/internal/logging"
	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/session"
	"byteflow-studio/internal/workflow"
	"byteflow-studio/services"

	// Import block packages so their init() functions register with the processing registry.
	_ "byteflow-studio/internal/analysis"
	"byteflow-studio/internal/input"
	_ "byteflow-studio/internal/processing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := logging.New()
	logger.Info("starting ByteFlow Studio")

	input.SetLogger(logger)

	// Open a transient in-memory store for the default session.
	// On load/save, a file-based store is used instead.
	workflowStore, err := workflow.OpenStore(":memory:")
	if err != nil {
		log.Fatalf("open workflow store: %v", err)
	}
	defer workflowStore.Close()

	sessionStore := session.NewStore(workflowStore.DB())

	defaultCfg := session.SessionConfig{StoreRaw: true, StoreProcessed: true, MaxSessions: 10}
	sessionMgr := session.NewManager(sessionStore, "default", defaultCfg, logger)

	engine := pipeline.NewEngine(logger)

	wfService := services.NewWorkflowService(workflowStore, logger)
	pipelineSvc := services.NewPipelineService(engine, sessionMgr, wfService, logger)
	sessionSvc := services.NewSessionService(sessionMgr, sessionStore, wfService, logger)

	wfService.SetEngine(pipelineSvc)
	sessionSvc.SetPipelineService(pipelineSvc)

	app := application.New(application.Options{
		Name:        "ByteFlow Studio",
		Description: "Visual serial data processing studio",
		Services: []application.Service{
			application.NewService(wfService),
			application.NewService(pipelineSvc),
			application.NewService(sessionSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	pipelineSvc.SetApp(app)
	sessionSvc.SetApp(app)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "ByteFlow Studio",
		Width:            1280,
		Height:           800,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
