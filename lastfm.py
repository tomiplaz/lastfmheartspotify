from sys import exit
from urllib.parse import urlunparse
from urllib.request import Request, urlopen
from urllib.error import HTTPError
from json import loads as jsonloads
from env import last_fm_api_key

class LastFm:
    scheme = 'http'
    netloc = 'ws.audioscrobbler.com'
    path = '/2.0/'

    def __init__(self, user):
        self.user = user
        self.common_params = {
            'api_key': last_fm_api_key,
            'user': user,
            'format': 'json',
        }
        self.common_headers = {
            'Accept': 'application/json'
        }

    def _get_params_string(self, params_dict):
        return '&'.join(['='.join((k, v)) for k, v in params_dict.items()])

    def _get_url(self, params):
        return urlunparse((
            LastFm.scheme,
            LastFm.netloc,
            LastFm.path,
            '',
            self._get_params_string({
                **self.common_params,
                **params,
            }),
            ''
        ))

    def get_loved_tracks(self):
        method = '.'.join(('user', 'getlovedtracks'))
        page = 0
        pages = 1
        items = []

        while page < pages:
            page += 1
            url = self._get_url({ 'method': method, 'page': str(page) })
            request = Request(url=url, headers=self.common_headers)

            try:
                response = urlopen(request)
                data = jsonloads(response.read())['lovedtracks']
                items = items + [(x['artist']['name'], x['name']) for x in data['track']]
                attrs = data['@attr']
                pages = int(attrs['totalPages'])
                print(str(len(items)) + '/' + attrs['total'])
            except HTTPError as e:
                print('\n'.join((str(e), jsonloads(e.read())['message'], url)))
                exit()

        return items
