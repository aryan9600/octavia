# Octavia

Octavia is a set of services that combine together to help with democracy based playlists.
Often, there have been times that we fight over which song should be played. The solution is to use **democracy**. :heart:

People can upvote their favourite songs, and Octavia helps keep a track of them and plays the most upvoted song automatically.
It uses the Spotify API to play songs and get recommendations along with Cloud Firestore to store songs and related details.

## Services:
- **Database trigger:** This trigger helps keep a track of the most upvoted song at all times. It is written in TypeScript and deployed on Firebase :fire:
- **Microservice:** The purpose of this microservice is to fetch the most upvoted song and play that song through Spotify. It is written in Python :snake: and uses Redis for caching. It is deployed on AWS ECS.
- **Cronjobs:** There are two cronjobs written in GoLang. One cronjob fetches new recommendations based on the top 5 upvoted songs. The other cronjob maintains a list of recently played songs and adds them back for voting. Both cronjobs are deployed as AWS Lambda fucntions using CloudWatch :cloud::watch:
