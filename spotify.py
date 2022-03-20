from sys import exit
from urllib.parse import urlunparse
from urllib.request import Request, urlopen
from urllib.error import HTTPError
from json import loads as jsonloads
from base64 import b64encode
from env import spotify_client_id, spotify_client_secret

class Spotify:
    scheme = 'https'
    netloc = 'accounts.spotify.com'
    client_id = ''
    client_secret = ''

    def __init__(self, user):
        self.user = user
        self.common_headers = {
            'Accept': 'application/json'
        }

    def _get_params_string(self, params_dict):
        return '&'.join(['='.join((k, v)) for k, v in params_dict.items()])

    def _get_url(self, path, params):
        return urlunparse((
            Spotify.scheme,
            Spotify.netloc,
            '/api' + path,
            '',
            self._get_params_string(params),
            ''
        ))

    def get_access_token(self):
        params = {
            'grant_type': 'client_credentials'
        }
        headers = {
            **self.common_headers,
            'Authorization': 'Basic ' +
                str(b64encode(str(spotify_client_id + ':' + spotify_client_secret).encode('utf-8')))
        }
        url = self._get_url('/token', params)
        request = Request(url=url, headers=headers, method='POST')

        try:
            response = urlopen(request)
            data = jsonloads(response.read())['lovedtracks']
            print(data)
        except HTTPError as e:
            print('\n'.join((str(e), str(jsonloads(e.read())), url)))
            exit()
