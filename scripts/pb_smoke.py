"""End-to-end smoke test for the SQLite single-node API paths touched by the
MySQL/cluster hardening work: auth collections (email uniqueness), multi-valued
fields (required-JSON columns) and partial unique indexes.

Usage: python3 scripts/pb_smoke.py [baseURL]
"""
import json
import sys
import time
import urllib.error
import urllib.request

BASE = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:38111"
SUF = str(int(time.time()))[-6:]
WIDGETS = "widgets" + SUF
CUSTOMERS = "customers" + SUF

failures = []


def check(label, got, want):
    ok = got == want
    print(("PASS " if ok else "FAIL ") + f"{label}: got {got}, want {want}")
    if not ok:
        failures.append(label)


def req(method, path, body=None, token=None):
    data = json.dumps(body).encode() if body is not None else None
    r = urllib.request.Request(BASE + path, data=data, method=method)
    r.add_header("Content-Type", "application/json")
    if token:
        r.add_header("Authorization", token)
    try:
        with urllib.request.urlopen(r, timeout=20) as resp:
            raw = resp.read()
            return resp.status, (json.loads(raw) if raw else {})
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {"raw": raw.decode()[:400]}


st, d = req("POST", "/api/admins/auth-with-password",
            {"identity": "admin@example.com", "password": "Sup3rSecret"})
assert st == 200, d
tok = d["token"]
print("login ok")

# 1. base collection: multi-select + json + partial unique index
st, d = req("POST", "/api/collections", {
    "name": WIDGETS, "type": "base", "displayName": "Widgets",
    "schema": [
        {"name": "title", "type": "text"},
        {"name": "colors", "type": "select",
         "options": {"maxSelect": 2, "values": ["red", "blue"]}},
        {"name": "meta", "type": "json"},
    ],
    "indexes": [
        f"CREATE UNIQUE INDEX idx_{WIDGETS}_title ON {WIDGETS} (title) WHERE title != ''"
    ],
}, tok)
check("create base collection", st, 200)
if st != 200:
    print(json.dumps(d)[:500])
    sys.exit(1)
wid = d["id"]

# 2. auth collection -> email uniqueness index path
st, d = req("POST", "/api/collections", {
    "name": CUSTOMERS, "type": "auth", "displayName": "Customers",
    "schema": [{"name": "nick", "type": "text"}],
}, tok)
check("create auth collection", st, 200)
if st != 200:
    print(json.dumps(d)[:500])
    sys.exit(1)

# 3. add a MULTI-RELATION field -> required-JSON column path on schema update
st, d = req("GET", "/api/collections/" + WIDGETS, None, tok)
coll = dict(d)
coll["schema"].append({
    "name": "friends", "type": "relation",
    "options": {"collectionId": wid},
})
st, d = req("PATCH", "/api/collections/" + WIDGETS, coll, tok)
check("add multi-relation field", st, 200)
if st != 200:
    print(json.dumps(d)[:500])

# 4. several records with EMPTY email must coexist
for i in range(2):
    st, d = req("POST", "/api/collections/" + CUSTOMERS + "/records", {
        "email": "", "password": "Sup3rSecret1", "passwordConfirm": "Sup3rSecret1",
    }, tok)
    check(f"auth record #{i} with empty email", st, 200)
    if st != 200:
        print(json.dumps(d)[:300])

# 5. create a record with multi-value select + json
st, d = req("POST", "/api/collections/" + WIDGETS + "/records", {
    "title": "w1", "colors": ["red", "blue"], "meta": {"a": 1},
}, tok)
check("create widget record", st, 200)
if st != 200:
    print(json.dumps(d)[:300])
    sys.exit(1)
rid = d["id"]

# 6. duplicate non-empty title must violate the partial unique index
st, _ = req("POST", "/api/collections/" + WIDGETS + "/records", {"title": "w1", "colors": []}, tok)
check("duplicate title rejected", st, 400)

# 7. blank titles must be allowed repeatedly
st, _ = req("POST", "/api/collections/" + WIDGETS + "/records", {"title": "", "colors": ["blue"]}, tok)
check("second blank title accepted", st, 200)

# 8. round-trip array + json values
st, d = req("GET", f"/api/collections/{WIDGETS}/records/{rid}", None, tok)
rec = d if "id" in d else d.get("data", {})
check("read back status", st, 200)
check("multi-select round-trip", rec.get("colors"), ["red", "blue"])
check("json round-trip", rec.get("meta"), {"a": 1})

# 9. malformed index expression must be rejected without dropping existing indexes
st, d = req("PATCH", "/api/collections/" + WIDGETS, dict(coll, indexes=["CREATE INDEX ON x (y)"]), tok)
check("malformed index expression rejected", st, 400)

st, d = req("GET", "/api/collections/" + WIDGETS + "/records?filter=(title='w1')", None, tok)
check("records still queryable after rejected index update", st, 200)

print()
if failures:
    print(f"{len(failures)} FAILURE(S): " + ", ".join(failures))
    sys.exit(1)
print("ALL SMOKE CHECKS PASSED")
