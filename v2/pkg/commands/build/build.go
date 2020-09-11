package build

import (
	"fmt"
	"os"
	"runtime"

	"github.com/leaanthony/slicer"
	"github.com/wailsapp/wails/v2/internal/logger"
	"github.com/wailsapp/wails/v2/internal/project"
)

// Mode is the type used to indicate the build modes
type Mode int

const (
	// Debug mode
	Debug Mode = iota
	// Production mode
	Production
)

var modeMap = []string{"Debug", "Production"}

// Options contains all the build options as well as the project data
type Options struct {
	LDFlags     string           // Optional flags to pass to linker
	Logger      *logger.Logger   // All output to the logger
	OutputType  string           // EG: desktop, server....
	Mode        Mode             // release or debug
	ProjectData *project.Project // The project data
	Pack        bool             // Create a package for the app after building
	Platform    string           // The platform to build for
	Compiler    string           // The compiler command to use
}

// GetModeAsString returns the current mode as a string
func GetModeAsString(mode Mode) string {
	return modeMap[mode]
}

// Build the project!
func Build(options *Options) (string, error) {

	// Extract logger
	outputLogger := options.Logger

	// Create a default logger if it doesn't exist
	if outputLogger == nil {
		outputLogger = logger.New()
	}

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Check platform
	validPlatforms := slicer.String([]string{"linux", "darwin"})
	if !validPlatforms.Contains(options.Platform) {
		return "", fmt.Errorf("platform %s not supported", options.Platform)
	}

	// Load project
	projectData, err := project.Load(cwd)
	if err != nil {
		return "", err
	}
	options.ProjectData = projectData

	// Save the project type
	projectData.OutputType = options.OutputType

	// Create builder
	var builder Builder

	switch projectData.OutputType {
	case "desktop":
		builder = newDesktopBuilder()
	case "hybrid":
		builder = newHybridBuilder()
	case "server":
		builder = newServerBuilder()
	default:
		return "", fmt.Errorf("cannot build assets for output type %s", projectData.OutputType)
	}

	// Set up our clean up method
	defer builder.CleanUp()

	// Initialise Builder
	builder.SetProjectData(projectData)

	outputLogger.Writeln("  - Building Wails Frontend")
	err = builder.BuildFrontend(outputLogger)
	if err != nil {
		return "", err
	}

	// Build the base assets
	outputLogger.Writeln("  - Compiling Assets")
	err = builder.BuildRuntime(options)
	if err != nil {
		return "", err
	}
	err = builder.BuildAssets(options)
	if err != nil {
		return "", err
	}

	// Compile the application
	outputLogger.Write("  - Compiling Application in " + GetModeAsString(options.Mode) + " mode...")
	err = builder.CompileProject(options)
	if err != nil {
		return "", err
	}
	outputLogger.Writeln("done.")
	// Do we need to pack the app?
	if options.Pack {

		outputLogger.Writeln("  - Packaging Application")

		// TODO: Allow cross platform build
		err = packageProject(options, runtime.GOOS)
		if err != nil {
			return "", err
		}
	}

	return projectData.OutputFilename, nil

}
