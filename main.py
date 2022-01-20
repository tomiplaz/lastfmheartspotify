from lastfm import LastFm
from user import lastfm_user, spotify_user

lastfm = LastFm(lastfm_user)

loved_tracks = lastfm.get_loved_tracks()
