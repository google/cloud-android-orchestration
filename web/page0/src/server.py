from absl import app as absl_app
from absl import flags

import flask
import requests

import test_apis

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
    resp = requests.get(_ANGULAR_URL.value)
    return flask.Response(resp.content, resp.status_code, resp.raw.headers.items())


@app.route("/<path:path>")
def proxy(path):
    resp = requests.get(f"{_ANGULAR_URL.value}{path}")
    return flask.Response(resp.content, resp.status_code, resp.raw.headers.items())


def main(argv):
  app.run(host='::', port=_PORT.value, debug=True)

if __name__ == '__main__':
  absl_app.run(main)
