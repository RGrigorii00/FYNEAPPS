package main

import (
	"FYNEAPPSSERVER/api/database"
	"FYNEAPPSSERVER/api/handlers"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Routes
	api := e.Group("/api/v1")

	// Computer routes
	computerHandler := handlers.NewComputerHandler(db)
	api.GET("/computers", computerHandler.GetComputers)
	api.GET("/computers/:id", computerHandler.GetComputer)
	api.POST("/computers", computerHandler.CreateComputer)
	api.PUT("/computers/:id", computerHandler.UpdateComputer)
	api.DELETE("/computers/:id", computerHandler.DeleteComputer)

	// Processor routes
	processorHandler := handlers.NewProcessorHandler(db)
	api.GET("/processors", processorHandler.GetProcessors)
	api.GET("/processors/:id", processorHandler.GetProcessor)
	api.GET("/computers/:computer_id/processors", processorHandler.GetProcessorsByComputer)
	api.POST("/processors", processorHandler.CreateProcessor)
	api.PUT("/processors/:id", processorHandler.UpdateProcessor)
	api.DELETE("/processors/:id", processorHandler.DeleteProcessor)

	// Memory routes
	memoryHandler := handlers.NewMemoryHandler(db)
	api.GET("/memory", memoryHandler.GetMemory)
	api.GET("/memory/:id", memoryHandler.GetMemoryEntry)
	api.GET("/computers/:computer_id/memory", memoryHandler.GetMemoryByComputer)
	api.POST("/memory", memoryHandler.CreateMemoryEntry)
	api.PUT("/memory/:id", memoryHandler.UpdateMemoryEntry)
	api.DELETE("/memory/:id", memoryHandler.DeleteMemoryEntry)

	// Network routes
	networkHandler := handlers.NewNetworkHandler(db)
	api.GET("/network-adapters", networkHandler.GetNetworkAdapters)
	api.GET("/network-adapters/:id", networkHandler.GetNetworkAdapter)
	api.GET("/computers/:computer_id/network-adapters", networkHandler.GetNetworkAdaptersByComputer)
	api.POST("/network-adapters", networkHandler.CreateNetworkAdapter)
	api.PUT("/network-adapters/:id", networkHandler.UpdateNetworkAdapter)
	api.DELETE("/network-adapters/:id", networkHandler.DeleteNetworkAdapter)

	// Disk routes
	diskHandler := handlers.NewDiskHandler(db)
	api.GET("/disks", diskHandler.GetDisks)
	api.GET("/disks/:id", diskHandler.GetDisk)
	api.GET("/computers/:computer_id/disks", diskHandler.GetDisksByComputer)
	api.POST("/disks", diskHandler.CreateDisk)
	api.PUT("/disks/:id", diskHandler.UpdateDisk)
	api.DELETE("/disks/:id", diskHandler.DeleteDisk)

	// Software routes

	softwareHandler := handlers.NewSoftwareHandler(db)
	updatesHandler := handlers.NewSoftwareUpdatesHandler(db)
	depsHandler := handlers.NewSoftwareDependenciesHandler(db)

	api.GET("/software", softwareHandler.GetSoftware)
	api.GET("/software/:id", softwareHandler.GetSoftwareByID)
	api.POST("/software", softwareHandler.CreateSoftware)
	api.PUT("/software/:id", softwareHandler.UpdateSoftware)
	api.DELETE("/software/:id", softwareHandler.DeleteSoftware)
	api.GET("/computers/:computer_id/software", softwareHandler.GetSoftwareByComputer)

	// Software updates routes
	api.GET("/updates", updatesHandler.GetUpdates)
	api.GET("/updates/:id", updatesHandler.GetUpdateByID)
	api.POST("/updates", updatesHandler.CreateUpdate)
	api.PUT("/updates/:id", updatesHandler.UpdateUpdate)
	api.DELETE("/updates/:id", updatesHandler.DeleteUpdate)
	api.GET("/software/:software_id/updates", updatesHandler.GetUpdatesBySoftware)

	// Software dependencies routes
	api.GET("/dependencies", depsHandler.GetDependencies)
	api.GET("/dependencies/:id", depsHandler.GetDependencyByID)
	api.POST("/dependencies", depsHandler.CreateDependency)
	api.PUT("/dependencies/:id", depsHandler.UpdateDependency)
	api.DELETE("/dependencies/:id", depsHandler.DeleteDependency)
	api.GET("/software/:software_id/dependencies", depsHandler.GetDependenciesBySoftware)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
