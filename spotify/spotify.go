package spotify

import (
	"io/ioutil"
	"log"
	"os"

	"code.google.com/p/portaudio-go/portaudio"
	sp "github.com/op/go-libspotify/spotify"
)

type audio struct {
	format sp.AudioFormat
	frames []byte
}

type audio2 struct {
	format sp.AudioFormat
	frames []int16
}

type portAudio struct {
	buffer chan *audio
}

func newPortAudio() *portAudio {
	return &portAudio{
		buffer: make(chan *audio, 8),
	}
}

var session *sp.Session

var (
	PlaylistsMap = make(map[string]*sp.Playlist)
)

// var player *sp.Player

func Initialise(initialised chan string, toPlay chan sp.Track) {
	appKey, err := ioutil.ReadFile("spotify_appkey.key")
	if err != nil {
		log.Fatal(err)
	}

	credentials := sp.Credentials{
		Username: "fabiofalci",
		Password: os.Getenv("SPOTIFY_PASSWORD"),
	}

	portaudio.Initialize()
	defer portaudio.Terminate()

	pa := newPortAudio()
	go pa.player()

	session, err = sp.NewSession(&sp.Config{
		ApplicationKey:   appKey,
		ApplicationName:  "testing",
		CacheLocation:    "tmp",
		SettingsLocation: "tmp",
		AudioConsumer:    pa,
	})

	if err != nil {
		log.Fatal(err)
	}

	// go func() {
	// 	for msg := range session.LogMessages() {
	// 		log.Print(msg)
	// 	}
	// }()

	if err = session.Login(credentials, false); err != nil {
		log.Fatal(err)
	}

	select {
	case err := <-session.LoginUpdates():
		if err != nil {
			log.Fatal(err)
		}
	}

	if session != nil {
		playlists, _ := session.Playlists()
		playlists.Wait()
		for i := 0; i < playlists.Playlists(); i++ {
			playlist := playlists.Playlist(i)
			playlist.Wait()

			if playlists.PlaylistType(i) == sp.PlaylistTypePlaylist {
				PlaylistsMap[playlist.Name()] = playlist
			}
		}
	}

	initialised <- ""

	for {
		select {
		case track := <-toPlay:
			Play(&track)
		}
	}
}

func Play(track *sp.Track) {
	player := session.Player()
	if err := player.Load(track); err != nil {
		println("error")
		log.Fatal(err)
	}
	player.Play()
}

func GetSession() *sp.Session {
	return session
}

func (pa *portAudio) player() {
	out := make([]int16, 2048*2)

	stream, err := portaudio.OpenDefaultStream(
		0,
		2,     // audio.format.Channels,
		44100, // float64(audio.format.SampleRate),
		len(out),
		&out,
	)
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	stream.Start()
	defer stream.Stop()

	// Decode the incoming data which is expected to be 2 channels and
	// delivered as int16 in []byte, hence we need to convert it.
	for audio := range pa.buffer {
		if len(audio.frames) != 2048*2*2 {
			panic("unexpected")
		}

		j := 0
		for i := 0; i < len(audio.frames); i += 2 {
			out[j] = int16(audio.frames[i]) | int16(audio.frames[i+1])<<8
			j++
		}

		stream.Write()
	}
}

func (pa *portAudio) WriteAudio(format sp.AudioFormat, frames []byte) int {
	audio := &audio{format, frames}
	// println("audio", len(frames), len(frames)/2)

	if len(frames) == 0 {
		// println("no frames")
		return 0
	}

	select {
	case pa.buffer <- audio:
		// println("return", len(frames))
		return len(frames)
	default:
		// println("buffer full")
		return 0
	}
}
