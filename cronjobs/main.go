package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	firestore "cloud.google.com/go/firestore"
	"github.com/joho/godotenv"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/api/iterator"

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

type FirestoreDeets struct {
	Type          string `json:"type"`
	ProjectID     string `json:"project_id"`
	PrivateKeyID  string `json:"private_key_id"`
	PrivateKey    string `json:"private_key"`
	ClientID      string `json:"client_id"`
	ClientEmail   string `json:"client_email"`
	AuthURI       string `json:"auth_uri"`
	TokenURI      string `json:"token_uri"`
	AuthProvider  string `json:"auth_provider_x509_cert_url"`
	ClientCertURL string `json:"client_x509_cert_url"`
}

func main() {
	initEnv()
	//SpotifyID := os.Getenv("SPOTIFY_ID")
	//SpotifySecret := os.Getenv("SPOTIFY_SECRET")
	//spotifyClient := setupSpotify(SpotifyID, SpotifySecret)
	firestoreClient := setupFirestore()
	//RefreshPlaylist(spotifyClient, firestoreClient)
	RestoreSongs(firestoreClient)
	fmt.Println(time.Now())
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
	//	PID := os.Getenv("PROJECT_ID")
	//	PKI := os.Getenv("PRIVATE_KEY_ID")
	//	PK := os.Getenv("PRIVATE_KEY")
	//	CEmail := os.Getenv("CLIENT_EMAIL")
	//	CId := os.Getenv("CLIENT_ID")
	//	Auth := os.Getenv("AUTH_URI")
	//	Token := os.Getenv("TOKEN_URI")
	//	AProvider := os.Getenv("AUTH_PROVIDER")
	//	ClientCert := os.Getenv("CLIENT_CERT")
	//fmt.Println(PK)
	//stuff := &FirestoreDeets{
	//	Type:          "service_account",
	//	ProjectID:     PID,
	//	PrivateKeyID:  PKI,
	//	PrivateKey:    PK,
	//	ClientEmail:   CEmail,
	//	ClientID:      CId,
	//	AuthURI:       Auth,
	//	TokenURI:      Token,
	//	AuthProvider:  AProvider,
	//	ClientCertURL: ClientCert,
	//}
	//deets, _ := json.Marshal(stuff)
	//fmt.Println(string(deets))
	file, err := os.Open("adminsdk.json")
	defer file.Close()
	values, _ := ioutil.ReadAll(file)
	opt := option.WithCredentialsJSON(values)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalln(err)
	}
	client, err := app.Firestore(context.Background())
	if err != nil {
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

func RestoreSongs(client *firestore.Client) {
	comparisonTime := time.Now().Add(time.Duration(-9) * time.Minute).UTC()
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
}
