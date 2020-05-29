package main

import (
	"context"
	firebase "firebase.google.com/go"
	"fmt"
	"google.golang.org/api/option"
	"log"
	"os"

	firestore "cloud.google.com/go/firestore"
	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
)

type SpotifyTrack struct {
	Name     string   `json:"name"`
	Artists  []string `json:"artists"`
	Duration int      `json:"duration"`
	Album    string   `json:"album"`
	Artwork  string   `json:"artwork"`
	TrackID  string   `json:"trackId"`
	Upvotes  int      `json:"upvotes"`
	URI      string   `json:"uri"`
}

func main() {
	initEnv()
	SpotifyID := os.Getenv("SPOTIFY_ID")
	SpotifySecret := os.Getenv("SPOTIFY_SECRET")
	spotifyClient := setupSpotify(SpotifyID, SpotifySecret)

	firestoreClient := setupFirestore()
	defer firestoreClient.Close()

	conn, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	psc := redis.PubSubConn{Conn: conn}
	psc.PSubscribe("__keyevent@0__:expired")
	for {
		switch msg := psc.Receive().(type) {
		case redis.Message:
			fmt.Printf("Message: %s %s\n", msg.Channel, msg.Data)
			UpdateSongAndStatus(firestoreClient, conn, spotifyClient)
			fmt.Printf("updated song and status")
		case redis.Subscription:
			fmt.Printf("Subscription: %s %s %d\n", msg.Kind, msg.Channel, msg.Count)
			if msg.Count == 0 {
				return
			}
		case error:
			fmt.Printf("error: %v\n", msg)
			return
		}
	}

}

func setupSpotify(id, secret string) spotify.Client {
	config := &clientcredentials.Config{
		ClientID:     id,
		ClientSecret: secret,
		TokenURL:     spotify.TokenURL,
	}
	token, err := config.Token(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	client := spotify.Authenticator{}.NewClient(token)
	return client
}

func setupFirestore() *firestore.Client {
	opt := option.WithCredentialsFile("adminsdk.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalln(err)
	}
	client, err := app.Firestore(context.Background())
	if err != nil{
		log.Fatalln(err)
	}
	return client
}

func initEnv() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

// UpdateSongAndStatus updates song and status.
func UpdateSongAndStatus(firestoreClient *firestore.Client, conn redis.Conn, spotifyClient spotify.Client) {
	var song SpotifyTrack
	var lastSong SpotifyTrack
	docsnap, err := firestoreClient.Doc("songs/mostUpvoted").Get(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	err = docsnap.DataTo(&song)
	if err != nil {
		log.Fatalln(err)
	}
	spotifyClient.PlayOpt(&spotify.PlayOptions{
		URIs: []spotify.URI{
			spotify.URI(song.URI),
		},
	})
	docsnap, err = firestoreClient.Doc("songs/nowPlaying").Get(context.Background())
	err = docsnap.DataTo(&lastSong)
	firestoreClient.Doc("recentSongs"+lastSong.URI).Create(context.Background(), lastSong)
	firestoreClient.Doc("songs/nowPlaying").Set(context.Background(), song)
	conn.Do("SETEX", song.URI, song.Duration/1000, song.Duration)
}
