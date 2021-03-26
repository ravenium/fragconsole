package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var recordingDir string
var recordingMode bool
var streamingMode bool
var srtStatusURL string
var srtStreamURL string
var srtStreamPassword string
var listenAddr string
var pollInterval int
var forwardMode bool
var reflectURL string
var processLaunchDelay int

// Stream struct - list of streams in the return from srtrelay
type Stream []struct {
	Name    string `json:"name"`
	Clients int    `json:"clients"`
	Created string `json:"created"`
}

// this will be our list of streams
var streams Stream

// inactive string for potential higher quality use later
//var ffBitcopyString = []string{"-c:v", "copy", "-c:a", "copy", "-hls_time", "1", "-hls_list_size", "60", "-hls_delete_threshold", "15", "-hls_flags", "delete_segments"}

func showVideoList(w http.ResponseWriter, req *http.Request) {

	var videoblock string
	for i := range streams {

		videoblock += ` <script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
	<BR> Stream Name: ` + streams[i].Name + `<BR>
	<video id="video` + strconv.Itoa(i) + `" controls></video>
	<script>
	  var video = document.getElementById('video` + strconv.Itoa(i) + `');
	  var videoSrc = '../` + streams[i].Name + `_srt.m3u8';
	  if (video.canPlayType('application/vnd.apple.mpegurl')) {
		video.src = videoSrc;
	  } else if (Hls.isSupported()) {
		var hls = new Hls();
		hls.loadSource(videoSrc);
		hls.attachMedia(video);
	  }
	</script>
	<P>
	`
	}
	fmt.Fprintf(w, videoblock)
	fmt.Fprintf(w, "<P>ACTIVE STREAMS:"+strconv.Itoa(len(streams)))
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func findStream(slice Stream, val string) bool {
	for i := range slice {
		if slice[i].Name == val {
			return true
		}
	}
	return false
}

func isRunningStream(slice map[string]bool, val string) bool {
	for i := range slice {
		if i == val {
			return true
		}
	}
	return false
}

func monitorStreams() {
	// Initialize maps for each potential ffmpeg process output
	sprocesses := make(map[string]*exec.Cmd)
	rprocesses := make(map[string]*exec.Cmd)
	fprocesses := make(map[string]*exec.Cmd)
	proctracker := make(map[string]bool)

	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Fatal("Cannot find ffmpeg!")
	}

	sltPath, err := exec.LookPath("srt-live-transmit")
	if err != nil {
		log.Fatal("Cannot find srt-live-transmit!")
	}

	// Poll every interval and launch/kill procs
	for {
		//sleep first so we can use this in continues
		time.Sleep(time.Duration(pollInterval) * time.Second)

		streamClient := http.Client{
			Timeout: time.Second * 2,
		}

		req, err := http.NewRequest(http.MethodGet, srtStatusURL, nil)
		if err != nil {
			log.Println("Cannot load SRT status URL: ", err)
			continue
		}

		req.Header.Set("User-Agent", "fragconsole-agent")
		res, getErr := streamClient.Do(req)

		if getErr != nil {
			log.Println("Cannot GET SRT status URL: ", getErr)
			continue
		}

		if res.Body != nil {
			defer res.Body.Close()
		}

		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			log.Fatal(readErr)
		}

		// assemble list of streams
		json.Unmarshal(body, &streams)

		log.Printf("Current live streams: %d \n", len(streams))

		//start new streams
		for i := 0; i < len(streams); i++ {
			if !isRunningStream(proctracker, streams[i].Name) {
				log.Println("New Stream found!", streams[i].Name)
				proctracker[streams[i].Name] = true
				if streamingMode {
					log.Println("Starting restream job for", streams[i].Name)
					sprocesses[streams[i].Name] = exec.Command(ffmpegPath, "-y", "-i", (srtStreamURL + "?streamid=play/" + streams[i].Name + srtStreamPassword), "-c:v", "libx264", "-x264opts", "keyint=1:no-scenecut", "-s", "640x360", "-r", "30", "-b:v", "900k", "-c:a", "aac", "-sws_flags", "bicubic", "-hls_time", "1", "-hls_list_size", "60", "-hls_delete_threshold", "15", "-hls_flags", "delete_segments", ("videos/" + streams[i].Name + "_srt.m3u8"))
					sprocesses[streams[i].Name].Start()
					//Wait a second so we don't overload ffmpeg launch
					time.Sleep(time.Second * time.Duration(processLaunchDelay))
				}
				if recordingMode {
					log.Println("Starting recording job for", streams[i].Name)
					t := time.Now().Format("20060102150405")
					rprocesses[streams[i].Name] = exec.Command(ffmpegPath, "-y", "-i", (srtStreamURL + "?streamid=play/" + streams[i].Name + srtStreamPassword), "-c:v", "copy", "-c:a", "copy", (recordingDir + "/" + streams[i].Name + "_" + t + ".mp4"))
					rprocesses[streams[i].Name].Start()
					//Wait a second so we don't overload ffmpeg launch
					time.Sleep(time.Second * time.Duration(processLaunchDelay))

				}
				if forwardMode {
					log.Println("Restreaming ", streams[i].Name, " to ", reflectURL)
					fprocesses[streams[i].Name] = exec.Command(sltPath, (srtStreamURL + "?streamid=play/" + streams[i].Name + srtStreamPassword), (reflectURL + "?streamid=publish/" + streams[i].Name + srtStreamPassword))
					fprocesses[streams[i].Name].Start()
					//Wait a second so we don't overload ffmpeg launch
					time.Sleep(time.Second * time.Duration(processLaunchDelay))
				}
			}
		}

		for j := range proctracker {
			if !findStream(streams, j) {
				log.Println("Removing ffmpeg processes for", j)
				delete(proctracker, j)
				if streamingMode {
					sprocesses[j].Process.Kill()
					sprocesses[j].Process.Wait()
					delete(sprocesses, j)
				}
				if recordingMode {
					rprocesses[j].Process.Kill()
					rprocesses[j].Process.Wait()
					delete(rprocesses, j)
				}
				if forwardMode {
					fprocesses[j].Process.Kill()
					fprocesses[j].Process.Wait()
					delete(fprocesses, j)
				}
			}
		}
	}
}

func main() {
	flag.StringVar(&srtStatusURL, "serverurl", "http://localhost:8080/streams", "URL of host running SRT status json endpoint")
	flag.StringVar(&srtStreamURL, "streamurl", "srt://localhost:1935", "URL of streaming server")
	flag.StringVar(&srtStreamPassword, "playpassword", "", "Password to play srt streams from srtrelay (optional)")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:3000", "Listen address for stream viewer")
	flag.BoolVar(&recordingMode, "r", false, "Record a copy of incoming streams.")
	flag.BoolVar(&streamingMode, "s", false, "Create a preview copy of incoming streams.")
	flag.StringVar(&recordingDir, "recpath", "record", "path for recordings (omit trailling slashes)")
	flag.IntVar(&pollInterval, "poll", 10, "Interval in seconds to poll for new SRT feeds.")
	flag.BoolVar(&forwardMode, "f", false, "Forward a copy of incoming streams to another SRT server.")
	flag.StringVar(&reflectURL, "reflect", "srt://localhost:1935", "SRT url to reflect all streams toward.  Usually used in lieu of recording or preview features.")
	flag.IntVar(&processLaunchDelay, "procdelay", 3, "Delay in seconds between ffmpeg/slt process launches.  Tweak if you are getting process exec failures.")
	flag.Parse()

	// if we have set a stream passsword, tack it on to the end of the ffmpeg strings above to properly ingest
	if srtStreamPassword != "" {
		srtStreamPassword = "/" + srtStreamPassword
	}

	//clear previous recordings from streaming dir
	os.RemoveAll("videos")

	if _, err := os.Stat("videos"); os.IsNotExist(err) {
		os.Mkdir("videos", 0755)
	}

	// throw in a blank file for the root video dir
	file, err := os.Create("videos/index.html")
	if err != nil {
		log.Fatal(err.Error())
	}
	file.Close()

	if _, err := os.Stat(recordingDir); os.IsNotExist(err) {
		os.Mkdir(recordingDir, 0755)
	}

	// Start thread for watching SRT server
	go monitorStreams()

	// Configure and launch http server
	fs := http.FileServer(http.Dir("./videos"))
	// Keeping videos dir served up as root.  If we want to have it as its own path, we should do:
	// prefixHandler := http.StripPrefix("/videos/", fs
	// http.Handle("/videos", prefixHandler)
	http.Handle("/", fs)
	http.HandleFunc("/monitor", showVideoList)
	log.Println("Starting Monitor web server at", listenAddr)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatalln("Error starting Web Server: ", err.Error())
	}
}
