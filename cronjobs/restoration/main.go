package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"google.golang.org/api/iterator"

	"log"
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
	lambda.Start(RestoreSongs)
}

func RestoreSongs() {

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

	// Restore songs
	comparisonTime := time.Now().Add(time.Duration(-60) * time.Minute).UTC()
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
		var track SpotifyTrack
		doc.DataTo(&track)
		fmt.Println(track.URI)
		client.Doc("songs/"+track.URI).Set(context.Background(), SpotifyTrack{
			Name:     track.Name,
			Album:    track.Name,
			Artwork:  track.Artwork,
			Duration: track.Duration,
			TrackID:  track.TrackID,
			Upvotes:  track.Upvotes,
			Artists:  track.Artists,
			Time:     track.Time,
		})
		doc.Ref.Delete(context.Background())
	}
	fmt.Printf("Done restoring songs!")
}
