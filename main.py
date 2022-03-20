from lastfm import LastFm
from spotify import Spotify
from user import lastfm_user, spotify_user

lastfm = LastFm(lastfm_user)
spotify = Spotify(spotify_user)

# loved_tracks = lastfm.get_loved_tracks()
spotify.get_access_token()
