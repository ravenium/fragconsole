- [What is this thing?](#what-is-this-thing)
- [Quick start:](#quick-start)
- [SRT Basics](#srt-basics)
- [Under the Hood](#under-the-hood)
- [Future Plans/TODO](#future-planstodo)
- [Errata](#errata)
- [Credits](#credits)

# What is this thing?

Fragconsole originated because of a need to visualize and preview streams in an [SRT](https://www.haivision.com/products/srt-secure-reliable-transport/) relay.  There are only a few open source solutions for SRT, and even fewer written that support multiple streamIDs on a single listening socket, are easy to use, and...well, are free.

Most of the need comes from running streams on Twitch for [Extra-Life](https://www.extra-life.org). In this use case, multiple streamers stream via SRT into a relay, from which a main OBS instance can then pick and choose which stream to grab, format, and rebroadcast to Twitch (which is still RTMP-only for most).


# Quick start:

- docker build -t srtrelay -f Dockerfile.srtrelay .

- docker build -t fragconsole -f Dockerfile.fragconsole .

- Edit docker-compose.yaml (sensible defaults are in there but YMMV)

- Optional: Download config.toml from voc/srtrelay, edit it, and map it to the srtrelay container (-v /path/to/config.toml:/conf/config.toml should do the trick)

- docker-compose up

- Push your favorite SRT streams to srt://your-docker-ip:1935?streamid=publish/yourcustomstreamid

- View streams at http://your-docker-ip:3000/monitor
- Pull streams via ffmpeg/obs/etc at srt://your-docker-ip:1935?streamid=play/yourcustomstreamid

# SRT Basics 

I won't go into SRT in detail as it's been better-covered elsewhere, but keep in mind a few key things:

- SRT is UDP-based, whereas RTMP is/was TCP.
- Each stream has a streamid, consisting of three parts:
  - A "verb", which in our use case is "publish" (send) or "play" (receive).
  - A stream name, such as "bob"
  - A stream password (totally optional).
- This ends up looking like streamid=publish/bob/withsomepassword, or just publish/bob if we don't want a password.  A person who wanted to play it would specify play/bob.

# Under the Hood

Fragconsole help options:

```
Usage:
  -f	Forward a copy of incoming streams to another SRT server in relay mode.
  -listen string
    	Listen address for stream viewer (default "127.0.0.1:3000")
  -playpassword string
    	Password to play srt streams from srtrelay (optional)
  -poll int
    	Interval in seconds to poll for new SRT feeds. (default 10)
  -procdelay int
    	Delay in seconds between ffmpeg/slt process launches.  Tweak if you are getting process exec failures. (default 3)
  -r	Record a copy of incoming streams.
  -recpath string
    	path for recordings (omit trailling slashes) (default "record")
  -reflect string
    	SRT url to reflect all streams toward.  Usually used in lieu of recording or preview features. (default "srt://localhost:1935").  Pair with -f.
  -s	Create a preview copy of incoming streams (via a web page).
  -serverurl string
    	URL of host running srtrelay API endpoint (default "http://localhost:8080/streams")
  -streamurl string
    	srt URL of streaming server (default "srt://localhost:1935")
```

- The docker-compose consists of two containers:

  - First is voc/srtrelay, which is the backbone of SRT streaming.  This container listens on one UDP port (I set it to 1935 for familiarity, but it can be anything you want) and one TCP port for the API (usually 8080). The API constantly publishes a list of connected publisher streams    
  - The second is Fragconsole (a work in progress name!), which essentially watches srtrelay by polling its API and taking action when it sees a new stream appear.  Currently there are three actions it can take:
    - Record:  The app uses ffmpeg to record the incoming stream without transcoding directly to a file.
    - Preview: The app uses ffmpeg to transcode the incoming stream to HLS format and adds it to the monitor web page. (Listens on 3000/TCP by default)
    - Relay: The app uses srt-live-transmit to re-send the incoming stream to another SRT server. This is especially useful if you have different streamers in different parts of the world and want to take advantage of "edge" streaming in regional data centers.

- Step by step:
  - Bob, a streamer, wants to send SRT so that Alice can receive it in her OBS, despite being behind firewalls and across the country. Alice has Fragconsole dockerized and running properly with both record and preview modes enabled!
  - Bob sends his OBS stream to srt://alice-server:1935?streamid=publish/bob.
  - The SRT stream is ingested by Alice's srtrelay container.
  - Fragconsole polls the srtrelay API and notes there is a new stream available.  It launches two ffmpeg processes, one each to transcode the stream to HLS, and one to record to disk.
  - Alice sees the stream in her web preview (http://alice-server:3000/monitor) and adds it to her OBS:  srt://alice-server:1935?streamid=play/bob
  - There are now three "play" streams running: 2 for ffmpeg, one for alice, for one "publish" stream for bob.

# Future Plans/TODO

- Document command line options in better detail.
- Implement live/notlive callbacks for things like tally/django.
- Better web page display output and adjustable HLS output quality.

# Errata

- If you like what you see, consider donating to your favorite Extra-Life streamer - help out a local Children's Hospital!
- In terms of resource usage, SRT is pretty low - I've seen south of 40% of a single CPU usage for ~30 streams (10 push, 20 pull).  
- ffmpeg is a kitchen sink swiss army knife and will eat your CPU alive, so keep an eye on your cores if you are both using preview and record as it will make 2 ffmpeg processes for each stream.
- VLC currently only supports SRT streamid divisions in the Nightlies of v4, in case you're trying to play things via VLC.
- The transcoding settings in the preview web page are intentionally not high quality so as to not overwhelm a browser with multiple streams at once.  You can change this by editing the ffmpeg line in the code directly for now.
- Average "reasonable" latency with most streamers looks to be about 5 seconds glass to glass. This beats the pants off of RTMP, but isn't quite as good as the sub 1 second of [OBS Ninja](https://obs.ninja), which uses WebRTC. However, note that SRT handles bad connections and latencies a lot better and is lower overhead.
- If you're seeing weird "snow" in your streams, your packets are probably exceeding timeouts for retransmit. Try increasing the latency setting in config.toml (general rule is 2.5x your RTT latency between streamer and server)

# Credits
- c3voc's [srtrelay](https://github.com/voc/srtrelay) for writing and maintaining a wonderful Golang implementation of the C bindings for SRT into a relay server), and for answering my 1001 questions.
- Edward Wu and [SRT-Live-Server](https://github.com/Edward-Wu/srt-live-server) for inspiring both of us that a simple SRT reflector was a thing.
- The many members of [Fragforce](https://fragforce.org) for putting up with endless hours of testing, tweaking, and feedback.  Come watch us stream!
