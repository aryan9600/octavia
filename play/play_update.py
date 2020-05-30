import redis
import spotipy
# import spotipy.util as util
from spotipy.oauth2 import SpotifyOAuth
import firebase_admin
from firebase_admin import credentials
from firebase_admin import firestore
import time


def updateSongAndDB(db, conn, client):
    doc = db.collection(u"songs").document(u"mostUpvoted").get()
    most_upvoted = doc.to_dict()
    client.start_playback(uris=[most_upvoted['uri']])
    doc = db.collection(u"songs").document(u"nowPlaying").get()
    last_song = doc.to_dict()
    if doc.exists is False:
        db.collection(u"songs").document(u"nowPlaying").create(most_upvoted)
        db.collection(u"songs").document(most_upvoted["uri"]).delete()
        return
    db.collection(u"songs").document(u"nowPlaying").set(most_upvoted)
    db.collection(u"songs").document(most_upvoted["uri"]).delete()
    db.collection(u"recentlyPlayed").document(last_song["uri"]).set({
        u'name': last_song['name'],
        u'duration': last_song['duration'],
        u'album': last_song['album'],
        u'artwork': last_song['artwork'],
        u'artists': last_song['artists'],
        u'upvotes': last_song['upvotes'],
        u'time': firestore.SERVER_TIMESTAMP,
        u'uri': last_song['uri'],
        u'trackId': last_song['trackId']
    })
    conn.setex(
        most_upvoted['uri'],
        # int(most_upvoted['duration']/1000),
        30,
        int(most_upvoted['duration']/1000)
    )


cred = credentials.Certificate('adminsdk.json')
firebase_admin.initialize_app(cred)

db = firestore.client()
scope = "user-read-playback-state,user-modify-playback-state"
username = '22ryi2ajn2xu5gzzdeuecrfni'
# token = util.prompt_for_user_token(username, scope)
sp = spotipy.Spotify(
        client_credentials_manager=SpotifyOAuth(scope=scope, username=username)
    )
# sp = spotipy.Spotify(auth=token)
r = redis.Redis(host="redis-server", port=6379, db=0)
p = r.pubsub()
p.psubscribe(b'__keyevent@0__:expired')
# p.psubscribe("*")
updateSongAndDB(db, r, sp)
while True:
    message = p.get_message()
    if message:
        if message['type'] == 'psubscribe':
            print('not acting')
        else:
            print('wasasaa')
            updateSongAndDB(db, r, sp)
    else:
        time.sleep(1)
