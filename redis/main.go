package main

import (
	"context"
	"fmt"
	"log"
	"os"

	firestore "cloud.google.com/go/firestore"
	api "github.com/aryan9600/octavia/trigger"
	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
)

func main() {
	initEnv()
	SpotifyID := os.Getenv("SPOTIFY_ID")
	SpotifySecret := os.Getenv("SPOTIFY_SECRET")
	setupSpotify(SpotifyID, SpotifySecret)
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
			UpdateSongAndStatus()
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

func initEnv() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

// UpdateSongAndStatus updates song and status.
func UpdateSongAndStatus(firestoreClient *firestore.Client, conn redis.Conn, spotifyClient *spotify.Client) {
	var song api.SpotifyTrack
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
	firestoreClient.Doc("songs/nowPlaying").Set(context.Background(), api.SpotifyTrack{
		Name:     song.Name,
		URI:      song.URI,
		Artists:  song.Artists,
		Album:    song.Album,
		Artwork:  song.Artwork,
		Upvotes:  song.Upvotes,
		TrackID:  song.TrackID,
		Duration: song.Duration,
	})
	conn.Do("SETEX", song.URI, song.Duration/1000, song.Duration)
}
