package trigger

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// FirestoreEvent is the payload of a Firestore event.
type FirestoreEvent struct {
	OldValue   FirestoreValue `json:"oldValue"`
	Value      FirestoreValue `json:"value"`
	UpdateMask struct {
		FieldPaths []string `json:"fieldPaths"`
	} `json:"updateMask"`
}

// FirestoreValue holds Firestore fields.
type FirestoreValue struct {
	CreateTime time.Time `json:"createTime"`
	Fields     MyData    `json:"fields"`
	Name       string    `json:"name"`
	UpdateTime time.Time `json:"updateTime"`
}

// MyData represents a value from Firestore.
type MyData struct {
	SpotifySong SpotifyTrack `json:"spotifyTrack"`
}

// GOOGLE_CLOUD_PROJECT is automatically set by the Cloud Functions runtime.
var projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")

// client is a Firestore client, reused between function invocations.
var client *firestore.Client

func init() {
	// opt := option.WithCredentialsFile("adminsdk.json")
	// app, err := firebase.NewApp(context.Background(), nil, opt)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	conf := &firebase.Config{ProjectID: projectID}

	// Use context.Background() because the app/client should persist across
	// invocations.
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		log.Fatalf("firebase.NewApp: %v", err)
	}
	client, err := app.Firestore(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()
}

// UpdateMostUpvoted is triggered by a change to a Firestore document.
// It keeps track of the most upvoted song.
func UpdateMostUpvoted(ctx context.Context, e FirestoreEvent) error {
	var song SpotifyTrack

	upvotes := e.Value.Fields.SpotifySong.Upvotes
	mostUpvoted := client.Doc("songs/mostUpvoted")

	docsnap, err := mostUpvoted.Get(ctx)
	if status.Code(err) == codes.NotFound {
		mostUpvoted.Create(ctx, SpotifyTrack{
			Name:     e.Value.Fields.SpotifySong.Name,
			Artists:  e.Value.Fields.SpotifySong.Artists,
			Album:    e.Value.Fields.SpotifySong.Album,
			Artwork:  e.Value.Fields.SpotifySong.Artwork,
			Duration: e.Value.Fields.SpotifySong.Duration,
			TrackID:  e.Value.Fields.SpotifySong.TrackID,
			Upvotes:  e.Value.Fields.SpotifySong.Upvotes,
			URI:      e.Value.Fields.SpotifySong.URI,
		})
		fmt.Println("create most upvoted")
		return nil
	}
	err = docsnap.DataTo(&song)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	if upvotes >= song.Upvotes {
		mostUpvoted.Set(ctx, SpotifyTrack{
			Name:     e.Value.Fields.SpotifySong.Name,
			Artists:  e.Value.Fields.SpotifySong.Artists,
			Album:    e.Value.Fields.SpotifySong.Album,
			Artwork:  e.Value.Fields.SpotifySong.Artwork,
			Duration: e.Value.Fields.SpotifySong.Duration,
			TrackID:  e.Value.Fields.SpotifySong.TrackID,
			Upvotes:  e.Value.Fields.SpotifySong.Upvotes,
		})
		fmt.Println("updated most upvoted")
	}
	return nil
}
