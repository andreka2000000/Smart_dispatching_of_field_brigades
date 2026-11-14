"""Behaviour checks against the installed native PyVRP, not a solver stub."""
import copy
import json
import subprocess
import sys
import threading
import unittest
import urllib.error
import urllib.request
from pathlib import Path
from unittest.mock import patch
from http.server import ThreadingHTTPServer
import server

def payload():
    root = Path("/fixtures")
    return {"reference": json.loads((root / "reference.json").read_text()),
            "day": json.loads((root / "day.json").read_text()),
            "matrix": json.loads((root / "matrix.json").read_text()),
            "budget_sec": 1.5, "seed": 0}

class SolverTests(unittest.TestCase):
    def test_profiles_exclusions_and_colocated_jobs(self):
        data = payload()
        forbidden = copy.deepcopy(data["day"]["orders"][0])
        forbidden.update(order_id="ORD-20261026-0099", requires_height=True)
        data["day"]["orders"].append(forbidden)
        for person in data["reference"]["staff"]:
            person["height_permit"] = False
        process = subprocess.run([sys.executable, "solve.py"], input=json.dumps(data),
                                 text=True, capture_output=True, timeout=4)
        self.assertEqual(process.returncode, 0, process.stderr)
        result = json.loads(process.stdout)
        self.assertEqual(result["version"], "0.14.0")
        self.assertEqual(len(result["candidates"]), 2)
        for candidate in result["candidates"]:
            ids = [oid for route in candidate["routes"] for oid in route["order_ids"]]
            self.assertEqual(len(ids), len(set(ids)))
            self.assertNotIn(forbidden["order_id"], ids)
            self.assertEqual(set(ids), {o["order_id"] for o in data["day"]["orders"][:2]})
            for route in candidate["routes"]:
                expected = data["reference"]["staff"][0 if data["day"]["orders"][0]["order_id"] in route["order_ids"] else 1]["staff_id"]
                self.assertEqual(route["staff_id"], expected)

class SupervisorTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.http = ThreadingHTTPServer(("127.0.0.1", 0), server.Handler)
        cls.thread = threading.Thread(target=cls.http.serve_forever, daemon=True)
        cls.thread.start()
        cls.url = "http://127.0.0.1:" + str(cls.http.server_port)
    @classmethod
    def tearDownClass(cls):
        cls.http.shutdown()
        cls.http.server_close()
        cls.thread.join()
    def post(self, body):
        request = urllib.request.Request(self.url + "/solve", data=json.dumps(body).encode(),
                                         headers={"Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(request, timeout=5) as response:
                return response.status, json.load(response)
        except urllib.error.HTTPError as response:
            return response.code, json.load(response)
    def test_health_and_bad_payload(self):
        with urllib.request.urlopen(self.url + "/health") as response:
            self.assertEqual(json.load(response)["version"], "0.14.0")
        for body in ([], None, {}, {"budget_sec": -1}, {"budget_sec": "bad"}):
            self.assertEqual(self.post(body)[0], 400)
    def test_busy_does_not_queue(self):
        server.SLOT.acquire()
        try:
            self.assertEqual(self.post(payload())[0], 503)
            with urllib.request.urlopen(self.url + "/health") as response:
                self.assertEqual(response.status, 200)
        finally:
            server.SLOT.release()
    def test_deadline_kills_child_and_releases_slot(self):
        with patch.object(server.subprocess, "Popen") as popen:
            child = popen.return_value
            child.communicate.side_effect = [subprocess.TimeoutExpired("solver", 1), (b"", b"")]
            child.poll.return_value = -9
            self.assertEqual(self.post(payload())[0], 504)
            child.kill.assert_called_once()
        self.assertTrue(server.SLOT.acquire(blocking=False))
        server.SLOT.release()
    def test_valid_http_solve(self):
        status, result = self.post(payload())
        self.assertEqual(status, 200, result)
        self.assertEqual(len(result["candidates"]), 2)

if __name__ == "__main__":
    unittest.main()
