from absl import app as absl_app
from absl import flags

import flask
import requests

import test_apis

EXCLUDED_HEADERS = {'content-encoding', 'content-length', 'transfer-encoding',
                    'connection'}

_PORT = flags.DEFINE_integer(
    'port',
    default=8071,
    help='default port',
)

_ANGULAR_URL = flags.DEFINE_string(
    'angular_url',
    default='http://localhost:4200/',
    help='default angular url',
)

app = flask.Flask(
    __name__
)

app.register_blueprint(test_apis.apis)

@app.route("/")
def index():
    response = requests.get(_ANGULAR_URL.value)

    headers = [(key, value) for (key, value) in response.raw.headers.items()
                 if key.lower() not in EXCLUDED_HEADERS]

    headers.append(("Access-Control-Allow-Origin", "*"))

    return flask.Response(response.content, response.status_code, headers)


@app.route("/<path:path>")
def proxy(path):
    response = requests.get(f"{_ANGULAR_URL.value}{path}")

    headers = [(key, value) for (key, value) in response.raw.headers.items()
                 if key.lower() not in EXCLUDED_HEADERS]

    headers.append(("Access-Control-Allow-Origin", "*"))

    return flask.Response(response.content, response.status_code, headers)

@app.after_request
def cors(response):
    header = response.headers
    header['Access-Control-Allow-Origin'] = '*'
    header['Access-Control-Allow-Methods'] =  "GET, POST, OPTIONS, PUT, DELETE"
    header['Access-Control-Allow-Headers'] =  "Content-Type"

    return response

def main(argv):
  app.run(host='::', port=_PORT.value, debug=True)

if __name__ == '__main__':
  absl_app.run(main)