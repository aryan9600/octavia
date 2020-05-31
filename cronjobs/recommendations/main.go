package main

import (
	"context"
	"fmt"

	firestore "cloud.google.com/go/firestore"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	Name     string    `json:"name"`
	Artists  []string  `json:"artists"`
	Duration int       `json:"duration"`
	Album    string    `json:"album"`
	Artwork  string    `json:"artwork"`
	TrackID  string    `json:"trackId"`
	Upvotes  int       `json:"upvotes"`
	URI      string    `json:"uri"`
	Time     time.Time `json:"time"`
}

func main() {
	lambda.Start(RefreshPlaylist)
}

func RefreshPlaylist() {

	// Spotify OAUTH stuff
	id := os.Getenv("SPOTIFY_ID")
	secret := os.Getenv("SPOTIFY_SECRET")
	config := &clientcredentials.Config{
		ClientID:     id,
		ClientSecret: secret,
		TokenURL:     spotify.TokenURL,
	}
	token, err := config.Token(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	cli := spotify.Authenticator{}.NewClient(token)

	// Firestore setup stuff
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("ap-south-1")},
	)

	downloader := s3manager.NewDownloader(sess)
	input := &s3.GetObjectInput{
		Bucket: aws.String("adminsdkjson"),
		Key:    aws.String("adminsdk.json"),
	}
	buf := aws.NewWriteAtBuffer([]byte{})
	downloader.Download(buf, input)
	fmt.Printf("Downloaded %v bytes", len(buf.Bytes()))
	fmt.Println(string(buf.Bytes()))

	opt := option.WithCredentialsJSON(buf.Bytes())
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalln(err)
	}
	client, err := app.Firestore(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	// Refresh the collection
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
	fmt.Println(mostUpvoted)
	result, err := cli.GetRecommendations(spotify.Seeds{Tracks: mostUpvoted}, nil, nil)
	if err != nil {
		log.Fatalln(err)
	}
	recommendations := result.Tracks
	for _, recommendation := range recommendations {
		song, err := cli.GetTrack(recommendation.ID)
		if err != nil {
			log.Fatalln(err)
		}
		var songArtists []string
		for _, artist := range song.SimpleTrack.Artists {
			songArtists = append(songArtists, artist.Name)
		}
		fmt.Println(song)
		docID := "songs/" + string(song.SimpleTrack.URI)
		client.Doc(docID).Create(context.Background(), SpotifyTrack{
			Name:     song.SimpleTrack.Name,
			Album:    song.Album.Name,
			Artwork:  song.Album.Images[0].URL,
			Duration: song.SimpleTrack.Duration,
			TrackID:  song.SimpleTrack.ID.String(),
			Upvotes:  0,
			URI:      string(song.SimpleTrack.URI),
			Artists:  songArtists,
			Time:     time.Now(),
		})
	}
}
