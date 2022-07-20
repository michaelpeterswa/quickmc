package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

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

	env := environment.LoadEnvironment()

	paperMCClient := papermc.NewPaperMCClient(&http.Client{Timeout: 10 * time.Second})
	buildNum, err := strconv.Atoi(env.Build)
	if err != nil {
		logger.Error("could not convert build number to int", zap.String("build", env.Build), zap.Error(err))
	}
	build, err := paperMCClient.GetBuild(env.Project, env.Version, buildNum)
	if err != nil {
		logger.Error("could not get build", zap.String("project", env.Project), zap.String("version", env.Version), zap.Int("build", buildNum), zap.Error(err))
	}
	link := papermc.GetDownloadLink(env.Project, env.Version, buildNum, build.Downloads.Application.Name)
	logger.Info("download link", zap.String("link", link))

	err = paperMCClient.Download(link, build.Downloads.Application.Sha256, "papermc.jar")
	if err != nil {
		logger.Error("could not download", zap.String("link", link), zap.String("sha256", build.Downloads.Application.Sha256), zap.Error(err))
	}

	err = os.Chmod("papermc.jar", 0755)
	if err != nil {
		logger.Error("could not chmod", zap.String("file", "papermc.jar"), zap.Error(err))
	}
	err = os.WriteFile("eula.txt", []byte("eula=true"), 0644)
	if err != nil {
		logger.Error("could not write eula", zap.Error(err))
	}

	zapWriter := zapio.Writer{
		Log:   logger,
		Level: zap.InfoLevel,
	}
	serverStartCmd := exec.Command("java", "-Xms2g", "-Xmx2g", "-jar", "papermc.jar", "--nogui")
	serverStartCmd.Stdout = &zapWriter
	err = serverStartCmd.Start()
	if err != nil {
		logger.Error("could not start server", zap.Error(err))
	}

	r := mux.NewRouter()
	r.HandleFunc("/healthcheck", handlers.HealthcheckHandler)
	r.Handle("/metrics", promhttp.Handler())
	http.Handle("/", r)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.Fatal("could not start http server", zap.Error(err))
	}
}
