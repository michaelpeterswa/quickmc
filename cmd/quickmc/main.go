package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/FragLand/minestat/Go/minestat"
	"github.com/gorilla/mux"
	"github.com/michaelpeterswa/quickmc/internal/environment"
	"github.com/michaelpeterswa/quickmc/internal/handlers"
	"github.com/michaelpeterswa/quickmc/internal/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
	"nw.codes/papermc-go/papermc"
)

func main() {
	logger, err := logging.InitZap()
	if err != nil {
		log.Panicf("could not acquire zap logger: %s", err.Error())
	}
	logger.Info("quickmc init...")

	if _, err := os.Stat("papermc.jar"); err != nil {
		env := environment.LoadEnvironment()

		paperMCClient := papermc.NewPaperMCClient(&http.Client{Timeout: 30 * time.Second})

		buildParams, err := GetLatestBuild(paperMCClient, env)
		if err != nil {
			logger.Fatal("could not get latest build")
		}
		build, err := paperMCClient.GetBuild(buildParams.Project, buildParams.Version, buildParams.BuildNum)
		if err != nil {
			logger.Fatal("could not get build", zap.String("project", env.Project), zap.String("version", env.Version), zap.Int("build", buildParams.BuildNum), zap.Error(err))
		}
		link := papermc.GetDownloadLink(build.ProjectID, build.Version, build.Build, build.Downloads.Application.Name)
		logger.Info("download link", zap.String("link", link))

		err = paperMCClient.Download(link, build.Downloads.Application.Sha256, "papermc.jar")
		if err != nil {
			logger.Fatal("could not download", zap.String("link", link), zap.String("sha256", build.Downloads.Application.Sha256), zap.Error(err))
		}

		err = os.Chmod("papermc.jar", 0755)
		if err != nil {
			logger.Fatal("could not chmod", zap.String("file", "papermc.jar"), zap.Error(err))
		}
		err = os.WriteFile("eula.txt", []byte("eula=true"), 0644)
		if err != nil {
			logger.Fatal("could not write eula", zap.Error(err))
		}
	}

	zapWriter := zapio.Writer{
		Log:   logger,
		Level: zap.InfoLevel,
	}
	serverStartCmd := exec.Command("java", "-Xms4g", "-Xmx4g", "-jar", "papermc.jar", "--nogui")
	serverStartCmd.Stdout = &zapWriter
	err = serverStartCmd.Start()
	if err != nil {
		logger.Fatal("could not start server", zap.Error(err))
	}

	go func() {
		time.Sleep(30 * time.Second)
		minestat.Init("localhost", "25565")
		fmt.Printf("Minecraft server status of %s on port %s:\n", minestat.Address, minestat.Port)
		if minestat.Online {
			fmt.Printf("Server is online running version %s with %s out of %s players.\n", minestat.Version, minestat.Current_players, minestat.Max_players)
			fmt.Printf("Message of the day: %s\n", minestat.Motd)
			fmt.Printf("Latency: %s\n", minestat.Latency)
		} else {
			fmt.Println("Server is offline!")
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/healthcheck", handlers.HealthcheckHandler)
	r.Handle("/metrics", promhttp.Handler())
	http.Handle("/", r)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.Fatal("could not start http server", zap.Error(err))
	}
}

type BuildParams struct {
	Project  string
	Version  string
	BuildNum int
}

func GetLatestBuild(pmcc *papermc.PaperMCClient, env *environment.Environment) (*BuildParams, error) {
	var buildParams = &BuildParams{}
	if env.Project == "" {
		buildParams.Project = "paper"
	} else {
		buildParams.Project = env.Project
	}
	if env.Version == "" {
		proj, err := pmcc.GetProject(buildParams.Project)
		if err != nil {
			return nil, err
		}
		buildParams.Version = proj.Versions[len(proj.Versions)-1]
	} else {
		buildParams.Version = env.Version
	}
	if env.Build == "" {
		builds, err := pmcc.GetVersion(buildParams.Project, buildParams.Version)
		if err != nil {
			return nil, err
		}
		buildParams.BuildNum = builds.Builds[len(builds.Builds)-1]
	} else {
		buildNum, err := strconv.Atoi(env.Build)
		if err != nil {
			return nil, err
		}
		buildParams.BuildNum = buildNum
	}

	return buildParams, nil
}
