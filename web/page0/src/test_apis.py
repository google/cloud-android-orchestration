"""APIs for test server.
"""
import flask

apis = flask.Blueprint('test_apis', __name__)


@apis.route('/test')
def testaaa():
  return 'this is test'
