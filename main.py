from lastfm import LastFm

lastfm = LastFm('tomivav')

loved_tracks = lastfm.getlovedtracks()

print(loved_tracks)
