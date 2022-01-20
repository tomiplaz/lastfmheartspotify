import sys
from urllib.parse import urlunparse
from urllib.request import Request, urlopen
from urllib.error import HTTPError
from json import loads as jsonloads

class LastFm:
    scheme = 'http'
    netloc = 'ws.audioscrobbler.com'
    path = '/2.0/'
    api_key = '9fd1cb3c8e24f418ac7b0166bfca3427'

    def __init__(self, user):
        self.user = user
        self.common_params = {
            'api_key': LastFm.api_key,
            'user': user,
            'format': 'json'
        }
        self.common_headers = {
            'Accept': 'application/json'
        }

    def _get_params_string(self, params_dict):
        return '&'.join(['='.join((k, v)) for k, v in params_dict.items()])

    def _get_method_url(self, method):
        return urlunparse((
            LastFm.scheme,
            LastFm.netloc,
            LastFm.path,
            '',
            self._get_params_string({
                **self.common_params,
                'method': method
            }),
            ''
        ))

    def getlovedtracks(self):
        url = self._get_method_url('.'.join(('user', 'getlovedtracks')))
        request = Request(url=url, headers=self.common_headers)

        try:
            response = urlopen(request)
            return jsonloads(response.read())
        except HTTPError as e:
            print('\n'.join((str(e), jsonloads(e.read())['message'], url)))
            sys.exit()
