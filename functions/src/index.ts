import * as functions from 'firebase-functions';
import * as admin from 'firebase-admin';
admin.initializeApp();
const db = admin.firestore();

exports.updateMostUpvoted = functions.firestore
    .document('songs/{songID}')
    .onUpdate(async (change, context) => {
        const song = change.after.data()
        await db.doc("songs/mostUpvoted").get().then(async doc => {
            if(!doc.exists){
                await db.doc("songs/mostUpvoted").create(song)
            } else {
                let mostUpvotes;
                // @ts-ignore
                mostUpvotes = doc.data().upvotes
                // @ts-ignore
                if (song.upvotes >= mostUpvotes) {
                    await db.doc("songs/mostUpvoted").set(song)
                }
            }
        })
    });

