"""APIs for test server.
"""
import flask
from collections import defaultdict
import json
import random
import string
import asyncio


def get_loop():
    try:
        loop = asyncio.get_running_loop()
        if loop.is_running():
            return loop
        return None
    except:
        return None


def bg_run(func, second):
    async def task():
        await asyncio.sleep(second)
        func()

    loop = get_loop()
    if loop:
        loop.create_task(task())
    else:
        asyncio.run(task())


apis = flask.Blueprint("test_apis", __name__)

mem = {"info": {"type": "cloud"}, "zones": defaultdict(list)}

DEFAULT_BUILD_SOURCE = {
    "android_ci_build_source": {
        "main_build": {
            "branch": "aosp-main",
            "build_id": "default",
            "target": "default",
        }
    },
}

DATA = {
    "zones": {
        "us-central1-a": [
            {
                "name": "us-host1",
                "gcp": {
                    "machine_type": "",
                    "min_cpu_platform": "",
                },
                "groups": [
                    {
                        "name": "group-1",
                        "cvds": [
                            {
                                "name": "cvd-1",
                                "build_source": DEFAULT_BUILD_SOURCE,
                                "status": "done",
                                "displays": [],
                                "group_name": "group-1",
                            },
                            {
                                "name": "cvd-2",
                                "build_source": DEFAULT_BUILD_SOURCE,
                                "status": "done",
                                "displays": [],
                                "group_name": "group-1",
                            },
                        ],
                    }
                ],
            },
            {
                "name": "us-host2",
                "gcp": {
                    "machine_type": "",
                    "min_cpu_platform": "",
                },
                "groups": [],
            },
        ],
        "ap-northeast2-c": [
            {
                "name": "ap-host1",
                "gcp": {
                    "machine_type": "",
                    "min_cpu_platform": "",
                },
                "groups": [],
            },
            {
                "name": "ap-host2",
                "gcp": {
                    "machine_type": "",
                    "min_cpu_platform": "",
                },
                "groups": [
                    {
                        "name": "group-1",
                        "cvds": [
                            {
                                "name": "cvd-1",
                                "build_source": DEFAULT_BUILD_SOURCE,
                                "status": "done",
                                "displays": [],
                                "group_name": "group-1",
                            },
                            {
                                "name": "cvd-2",
                                "build_source": DEFAULT_BUILD_SOURCE,
                                "status": "done",
                                "displays": [],
                                "group_name": "group-1",
                            },
                        ],
                    },
                    {
                        "name": "group-2",
                        "cvds": [
                            {
                                "name": "cvd-1",
                                "build_source": DEFAULT_BUILD_SOURCE,
                                "status": "done",
                                "displays": [],
                                "group_name": "group-2",
                            },
                            {
                                "name": "cvd-2",
                                "build_source": DEFAULT_BUILD_SOURCE,
                                "status": "done",
                                "displays": [],
                                "group_name": "group-2",
                            },
                        ],
                    },
                    {
                        "name": "group-3",
                        "cvds": [
                            {
                                "name": "cvd-1",
                                "build_source": DEFAULT_BUILD_SOURCE,
                                "status": "done",
                                "displays": [],
                                "group_name": "group-3",
                            },
                            {
                                "name": "cvd-2",
                                "build_source": DEFAULT_BUILD_SOURCE,
                                "status": "done",
                                "displays": [],
                                "group_name": "group-3",
                            },
                        ],
                    },
                ],
            },
        ],
    }
}


def gen_host_name(length):
    return "".join(random.choices(string.ascii_uppercase + string.digits, k=length))


def gen_operation_name(length):
    return "".join(random.choices(string.ascii_lowercase + string.digits, k=length))


def find_host(hosts, name):
    if not hosts:
        return None, -1

    for idx, host in enumerate(hosts):
        if host["name"] == name:
            return host, idx
    return None, -1


def find_group(groups, name):
    if not groups:
        return None, -1

    for idx, group in enumerate(groups):
        if group["name"] == name:
            return group, idx
    return None, -1


def init():
    for zone in DATA["zones"]:
        mem["zones"][zone] = DATA["zones"][zone]


# GET /info
@apis.route("/api/info", methods=["GET"])
def info():
    return mem["info"]


# GET /v1/zones
@apis.route("/api/v1/zones", methods=["GET"])
def zones():
    zones = list(mem["zones"].keys())
    return {"items": zones}


# GET /v1/zones/:zone/hosts
@apis.route("/api/v1/zones/<zone>/hosts", methods=["GET"])
def get_hosts(zone):
    hosts = []
    for host in mem["zones"][zone]:
        hosts.append({"name": host["name"], "gcp": host["gcp"]})
    return {"items": hosts}


# POST /v1/zones/:zone/hosts
@apis.route("/api/v1/zones/<zone>/hosts", methods=["POST"])
def post_host(zone):
    body = flask.request.json

    def task():
        mem["zones"][zone].append(
            {
                "name": gen_host_name(5),
                "gcp": body["host_instance"]["gcp"],
                "groups": [],
            }
        )

    bg_run(task, 1)

    return {"name": gen_operation_name(15), "done": False}


# DELETE /v1/zones/:zone/hosts/:host
@apis.route("/api/v1/zones/<zone>/hosts/<hostname>", methods=["DELETE"])
def delete_host(zone, hostname):
    _, idx = find_host(mem["zones"][zone], hostname)

    if idx < 0:
        flask.abort(404)

    def task():
        mem["zones"][zone].pop(idx)

    bg_run(task, 1)

    return {"name": gen_operation_name(15), "done": False}


# GET /v1/zones/:zone/hosts/:host/groups
@apis.route("/api/v1/zones/<zone>/hosts/<hostname>/groups", methods=["GET"])
def get_groups(zone, hostname):
    host, _ = find_host(mem["zones"][zone], hostname)
    if not host:
        flask.abort(404)

    return {"groups": host["groups"]}


# DELETE /v1/zones/:zone/hosts/:host/groups
@apis.route(
    "/api/v1/zones/<zone>/hosts/<hostname>/groups/<groupname>", methods=["DELETE"]
)
def delete_group(zone, hostname, groupname):
    host, hostidx = find_host(mem["zones"][zone], hostname)

    if not host:
        flask.abort(404)

    _, groupidx = find_group(host["groups"], groupname)

    if groupidx < 0:
        flask.abort(404)

    def task():
        mem["zones"][zone][hostidx]["groups"].pop(groupidx)

    bg_run(task, 10)

    return {"name": gen_operation_name(15), "done": False}


# POST /v1/zones/:zone/hosts/:host/cvds
@apis.route("/api/v1/zones/<zone>/hosts/<hostname>/cvds", methods=["POST"])
async def post_group(zone, hostname):
    host, hostidx = find_host(mem["zones"][zone], hostname)

    if not host:
        flask.abort(404)

    body = flask.request.json
    group = body["group"]

    def task():
        mem["zones"][zone][hostidx]["groups"].append(group)
        print(mem["zones"][zone][hostidx]["groups"])

    bg_run(task, 3)

    return {"name": gen_operation_name(15), "done": False}


"""
Mock server-only APIs 
"""


@apis.route("/api/reset", methods=["GET"])
def reset():
    init()


"""
Initialize memory
"""

init()
