// Photo Select: swipe through a huge photo library on your phone (or a TV via
// screen mirroring) and keep the ones you love. Runs on the computer that has
// the photos; phones on the same Wi-Fi open it in the browser.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
)

var version = "dev"

func main() {
	port := flag.Int("port", 8787, "port to listen on (the next free port is used if taken)")
	noBrowser := flag.Bool("no-browser", false, "don't open the setup page in the browser")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("photo-select", version)
		return
	}

	app := NewApp(defaultConfigPath(), NewPreviewer())
	app.Resume()

	ln, p, err := listen(*port)
	if err != nil {
		log.Fatalf("can't start server: %v", err)
	}
	srv := &Server{app: app, port: p}

	local := "http://localhost:" + strconv.Itoa(p)
	fmt.Println("")
	fmt.Println("  📸  Photo Select", version, "is running")
	fmt.Println("")
	fmt.Println("  On this computer:  ", local)
	for _, u := range phoneURLs(p) {
		fmt.Println("  On your phone:     ", u)
	}
	fmt.Println("")
	fmt.Println("  Phone and computer must be on the same Wi-Fi.")
	fmt.Println("  Keep this window open while you swipe. Press Ctrl+C to quit.")
	fmt.Println("")

	if !*noBrowser {
		openBrowser(local)
	}
	log.Fatal(http.Serve(ln, srv.Handler()))
}

func listen(port int) (net.Listener, int, error) {
	var err error
	for p := port; p < port+20; p++ {
		var ln net.Listener
		if ln, err = net.Listen("tcp", ":"+strconv.Itoa(p)); err == nil {
			return ln, p, nil
		}
	}
	return nil, 0, err
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
