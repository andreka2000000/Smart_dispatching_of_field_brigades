"""PyVRP proposes order sequences. Go owns the exact case constraints.

No Enterprise features are used. Two free VRPTW models are explored:
an optimistic whole shift and two trips separated by a fixed lunch interval.
Exact lunch placement, initial stock, restocking and schedule repair stay in Go.
"""
import json
import sys
import time
STARTED = time.monotonic()

import numpy as np
from pyvrp import Client, Depot, Location, ProblemData, VehicleType, Solution, SolveParams, PenaltyParams, solve
from pyvrp.stop import MaxRuntime
from datetime import datetime

VERSION = "0.14.0"

def seconds(value, midnight):
    return int((datetime.fromisoformat(value) - midnight).total_seconds())

def local(value):
    hour, minute = map(int, value.split(":"))
    return hour * 3600 + minute * 60

def equipment(value):
    return [value[k] for k in ("ONT", "ROUTER", "STB")]

def qualified(staff, order, work):
    return (bool(set(staff["profiles"]) & set(work["profiles"]))
            and order["technology"] in staff["technologies"]
            and (not order["requires_height"] or staff["height_permit"])
            and all(a <= b for a, b in zip(equipment(order["required_equipment"]), equipment(staff["equipment_capacity"]))))

def propose(payload):
    budget = float(payload["budget_sec"])
    deadline = STARTED + min(180.0, max(0.1, budget))
    reference, day, matrix = payload["reference"], payload["day"], payload["matrix"]
    if len(day["orders"]) > 600 or len(reference["staff"]) > 100:
        raise ValueError("optimizer resource limit: 600 orders / 100 staff")
    midnight = datetime.fromisoformat(day["date"] + "T00:00:00+05:00")
    now = seconds(day["now"], midnight)
    works = {w["code"]: w for w in reference["work_types"]}
    excluded = set(payload.get("excluded_order_ids", []))
    absent = set(day.get("absent_staff_ids", []))
    staff = sorted((s for s in reference["staff"] if s["staff_id"] not in absent
                    and max(local(s["shift"]["start"]), now) < local(s["shift"]["end"])), key=lambda s: s["staff_id"])
    orders = sorted((o for o in day["orders"] if o["order_id"] not in excluded
                     and seconds(o["latest_start"], midnight) >= now), key=lambda o: o["order_id"])
    if not staff or not orders:
        return {"version": VERSION, "candidates": [], "reason": "no_eligible_work"}
    centers = sorted(reference["service_centers"], key=lambda c: c["service_center_id"])
    depots = [Depot(i, name=c["service_center_id"]) for i, c in enumerate(centers)]
    depot_index = {c["service_center_id"]: i for i, c in enumerate(centers)}
    locations = [Location(c["lon"], c["lat"], name=c["service_center_id"]) for c in centers]
    # Every job has a distinct model location, including colocated subscribers:
    # eligibility can differ by work type at the same physical building.
    locations += [Location(o["lon"], o["lat"], name=o["order_id"]) for o in orders]
    source_index = {point: i for i, point in enumerate(matrix["points"])}
    indices = [source_index[c["service_center_id"]] for c in centers] + [source_index[o["building_id"]] for o in orders]
    travel = np.asarray(matrix["durations_sec"], dtype=np.int64)[np.ix_(indices, indices)]
    # Bounded search prizes favour coverage and repeats without overwhelming
    # feasibility penalties. Go applies the exact lexicographic acceptance rule.
    # "Distance" is intentionally a cost measured in travel seconds internally.
    weighted_prize, raw_prize = 50000, 1000
    clients = [Client(len(depots) + i, delivery=equipment(o["required_equipment"]),
                      service_duration=works[o["work_type_code"]]["duration_min"] * 60,
                      tw_early=max(now, seconds(o["slot"]["start"], midnight)),
                      tw_late=min(seconds(o["latest_start"], midnight), seconds(o["deadline"], midnight) - works[o["work_type_code"]]["duration_min"] * 60),
                      prize=(2 if o["is_repeat"] else 1) * weighted_prize + raw_prize,
                      required=False, name=o["order_id"])
               for i, o in enumerate(orders)]
    masks, profiles, staff_profile = {}, [], []
    for person in staff:
        mask = tuple(qualified(person, o, works[o["work_type_code"]]) for o in orders)
        if mask not in masks:
            distances = travel.copy()
            blocked = [len(depots) + i for i, allowed in enumerate(mask) if not allowed]
            # 86400 exceeds every permitted single-day shift. Feasible solutions
            # cannot traverse these edges, regardless of objective penalties.
            distances[blocked, :] = 86400
            distances[:, blocked] = 86400
            np.fill_diagonal(distances, 0)
            masks[mask] = len(profiles)
            profiles.append(distances)
        staff_profile.append(masks[mask])
    candidates = []
    for variant in ("whole_shift", "split_lunch"):
        remaining = deadline - time.monotonic() - 0.15
        if remaining < 0.2:
            break
        types, bindings = [], []
        for i, person in enumerate(staff):
            shift = person["shift"]
            start, end = max(now, local(shift["start"])), local(shift["end"])
            intervals = [(start, end)]
            if variant == "split_lunch":
                lunch_start = min(local(shift["break_window"]["to"]), local(shift["break_window"]["from"]) + (i % 4) * 3600)
                lunch_end = lunch_start + shift["break_duration_min"] * 60
                intervals = [(start, lunch_start), (max(start, lunch_end), end)]
            for segment, (early, late) in enumerate(intervals):
                if early >= late:
                    continue
                types.append(VehicleType(num_available=1, capacity=equipment(person["equipment_capacity"]),
                                         start_depot=depot_index[person["service_center_id"]],
                                         end_depot=depot_index[person["service_center_id"]],
                                         tw_early=early, tw_late=late, shift_duration=late-early,
                                         profile=staff_profile[i], name=person["staff_id"] + f"/{segment}"))
                bindings.append((person["staff_id"], segment))
        data = ProblemData(locations=locations, clients=clients, depots=depots, vehicle_types=types,
                           distance_matrices=profiles, duration_matrices=profiles)
        runtime = max(0.05, (deadline - time.monotonic() - 0.15) / (2 if variant == "whole_shift" else 1))
        params = SolveParams(penalty=PenaltyParams(solutions_between_updates=50, target_feasible=0.8,
                                                  min_penalty=100, max_penalty=1000000))
        result = solve(data, stop=MaxRuntime(runtime), seed=int(payload.get("seed", 0)), collect_stats=False,
                       display=False, params=params, initial_solution=Solution(data, []))
        if not result.is_feasible():
            continue
        segments = {}
        for route in result.best.routes():
            person_id, segment = bindings[route.vehicle_type()]
            order_ids = [orders[a.idx]["order_id"] for a in route if a.is_client()]
            segments.setdefault(person_id, []).append((segment, order_ids))
        routes = [{"staff_id": person_id, "order_ids": [oid for _, ids in sorted(parts) for oid in ids]}
                  for person_id, parts in sorted(segments.items())]
        candidates.append({"model": variant, "routes": routes, "proposed_visits": sum(len(r["order_ids"]) for r in routes)})
    return {"version": VERSION, "seed": int(payload.get("seed", 0)), "elapsed_ms": int((time.monotonic()-STARTED)*1000), "candidates": candidates}

if __name__ == "__main__":
    print(json.dumps(propose(json.load(sys.stdin)), separators=(",", ":")))
