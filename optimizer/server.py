"""Internal-only bounded worker. Each solve runs in a killable subprocess."""
import json
import subprocess
import sys
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from importlib.metadata import version

SLOT = threading.Lock()
class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass
    def reply(self, status, body):
        raw = json.dumps(body).encode()
        try:
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(raw)))
            self.end_headers()
            self.wfile.write(raw)
        except (BrokenPipeError, ConnectionResetError):
            pass
    def do_GET(self):
        if self.path != "/health":
            return self.reply(404, {"error": "not_found"})
        self.reply(200, {"status": "ok", "engine": "PyVRP", "version": version("pyvrp")})
    def do_POST(self):
        if self.path != "/solve":
            return self.reply(404, {"error": "not_found"})
        try:
            length = int(self.headers.get("Content-Length", "0"))
            if length <= 0 or length > 32 * 1024 * 1024:
                return self.reply(413, {"error": "body_limit"})
            self.connection.settimeout(10)
            payload = json.loads(self.rfile.read(length))
            budget = float(payload["budget_sec"])
            if not 0.1 <= budget <= 180:
                return self.reply(400, {"error": "invalid_budget"})
        except (ValueError, KeyError, TypeError, TimeoutError):
            return self.reply(400, {"error": "invalid_request"})
        if not SLOT.acquire(blocking=False):
            return self.reply(503, {"error": "worker_busy"})
        child = None
        try:
            child = subprocess.Popen([sys.executable, "solve.py"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            try:
                stdout, stderr = child.communicate(json.dumps(payload).encode(), timeout=budget + 0.75)
            except subprocess.TimeoutExpired:
                child.kill()
                child.communicate()
                return self.reply(504, {"error": "worker_deadline"})
            if child.returncode or len(stdout) > 1024 * 1024:
                print("Solver failure:", stderr[-2000:].decode(errors="replace"), file=sys.stderr, flush=True)
                return self.reply(422, {"error": "model_failed"})
            self.reply(200, json.loads(stdout))
        except (OSError, ValueError, TypeError):
            self.reply(503, {"error": "worker_failure"})
        finally:
            if child is not None and child.poll() is None:
                child.kill()
                child.wait()
            SLOT.release()

if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", 8090), Handler).serve_forever()
