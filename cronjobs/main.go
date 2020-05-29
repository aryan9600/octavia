package main

import (
	firestore "cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/api/iterator"
	"log"
	"os"
	"time"

	"google.golang.org/api/option"

	firebase "firebase.google.com/go"
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

type SpotifyTrackWithTime struct {
	Name     string   `json:"name"`
	Artists  []string `json:"artists"`
	Duration int      `json:"duration"`
	Album    string   `json:"album"`
	Artwork  string   `json:"artwork"`
	TrackID  string   `json:"trackId"`
	Upvotes  int      `json:"upvotes"`
	URI      string   `json:"uri"`
	Time	 time.Time		`json:time`
}

func main(){
	initEnv()
	SpotifyID := os.Getenv("SPOTIFY_ID")
	SpotifySecret := os.Getenv("SPOTIFY_SECRET")
	spotifyClient := setupSpotify(SpotifyID, SpotifySecret)
	firestoreClient := setupFirestore()
	RefreshPlaylist(spotifyClient, firestoreClient)
}

func initEnv() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
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

// RefreshPlaylist is a cronjob which gets new recommendations from the Spotify API, and then updates
// the playlist and the Firestore datbase with the new songs.
func RefreshPlaylist(cli spotify.Client, client *firestore.Client) {
	var mostUpvoted []spotify.ID
	iter := client.Collection("songs").OrderBy("upvotes", firestore.Desc).Limit(5).Documents(context.Background())
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return
		}
		var track SpotifyTrack
		doc.DataTo(&track)
		mostUpvoted = append(mostUpvoted, spotify.ID(track.TrackID))
	}
	result, err := cli.GetRecommendations(spotify.Seeds{Tracks: mostUpvoted}, nil, nil)
	if err!=nil{
		log.Fatalln(err)
	}
	recommendations := result.Tracks
	for _, recommendation := range recommendations {
		song, err := cli.GetTrack(recommendation.ID)
		if err!=nil{
			log.Fatalln(err)
		}
		var songArtists []string
		for _, artist := range song.SimpleTrack.Artists{
			songArtists = append(songArtists, artist.Name)
		}
		docID := "recentlyPlayed/"+string(song.SimpleTrack.URI)
		client.Doc(docID).Create(context.Background(), SpotifyTrackWithTime{
			Name: song.SimpleTrack.Name,
			Album: song.Album.Name,
			Artwork: song.Album.Images[0].URL,
			Duration: song.SimpleTrack.Duration,
			TrackID: song.SimpleTrack.ID.String(),
			Upvotes: 0,
			URI: string(song.SimpleTrack.URI),
			Artists: songArtists,
			Time: time.Now(),
		},)
	}
}

func RestoreSongs(client *firestore.Client) {
	comparisonTime := time.Now().Add(time.Duration(-90) * time.Minute)
	query := client.Collection("recentlyPlayed").Where("time", "<", comparisonTime)
	iter := query.Documents(context.Background())
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(doc.Data())
		var track SpotifyTrackWithTime
		doc.DataTo(&track)
		client.Doc("songs/"+track.URI).Set(context.Background(), SpotifyTrack{
			Name: track.Name,
			Album: track.Name,
			Artwork: track.Artwork,
			Duration: track.Duration,
			TrackID: track.TrackID,
			Upvotes: 0,
			Artists: track.Artists,
		})
		doc.Ref.Delete(context.Background())
	}
}


