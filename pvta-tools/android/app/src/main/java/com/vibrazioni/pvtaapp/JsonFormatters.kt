package com.vibrazioni.pvtaapp

import org.json.JSONArray
import org.json.JSONObject

object JsonFormatters {
    fun formatRoutes(json: String): List<String> {
        val routes = JSONArray(json)
        val lines = mutableListOf<String>()
        for (i in 0 until routes.length()) {
            val route = routes.getJSONObject(i)
            lines += "${route.optString("ShortName")} | ${route.optString("LongName")} | ${route.optInt("RouteId")}"
        }
        return lines.ifEmpty { listOf("No routes") }
    }

    fun formatVehicles(json: String): List<String> {
        val vehicles = JSONArray(json)
        if (vehicles.length() == 0) return listOf("No active vehicles")
        val lines = mutableListOf<String>()
        for (i in 0 until vehicles.length()) {
            val item = vehicles.getJSONObject(i)
            val vehicle = item.getJSONObject("vehicle")
            lines += buildList {
                add("${vehicle.optInt("VehicleId")} (${vehicle.optString("DirectionLong", "Unknown direction")})")
                add("Location: ${vehicle.optDouble("Latitude")}, ${vehicle.optDouble("Longitude")}")
                add("Last Stop: ${vehicle.optString("LastStop", "Unknown")}")
                item.optJSONObject("current_stop")?.let { add("Current Stop: ${it.optString("Name")}") }
                item.optJSONObject("next_stop")?.let { add("Next Stop: ${it.optString("Name")}") }
                add("Status: ${vehicle.optString("DisplayStatus", "Unknown")}")
                add("Deviation: ${vehicle.optInt("Deviation")} min")
                add("Occupancy: ${vehicle.optString("OccupancyStatusReportLabel", "Unknown")}")
            }.joinToString("\n")
            lines += ""
        }
        return lines.dropLastWhile { it.isBlank() }
    }

    fun formatRouteStatus(json: String): List<String> {
        val obj = JSONObject(json)
        val route = obj.getJSONObject("route")
        val lines = mutableListOf<String>()
        lines += "Route: ${route.optString("ShortName")} - ${route.optString("LongName")}"
        lines += ""
        lines += "Vehicles:"
        lines += formatVehicles(obj.getJSONArray("vehicles").toString())
        lines += ""
        lines += "Alerts:"
        val messages = obj.optJSONArray("messages") ?: JSONArray()
        if (messages.length() == 0) {
            lines += "none"
        } else {
            for (i in 0 until messages.length()) {
                val msg = messages.getJSONObject(i)
                lines += msg.optString("Header").ifBlank { msg.optString("Message", "Alert") }
            }
        }
        return lines
    }

    fun formatStops(json: String): List<String> {
        val stops = JSONArray(json)
        val lines = mutableListOf<String>()
        for (i in 0 until stops.length()) {
            val stop = stops.getJSONObject(i)
            lines += "${stop.optInt("StopId")} | ${stop.optString("Name")}"
        }
        return lines.ifEmpty { listOf("No stops") }
    }

    fun formatStopStatus(json: String): List<String> {
        val obj = JSONObject(json)
        val stop = obj.getJSONObject("stop")
        val lines = mutableListOf("Stop: ${stop.optString("Name")}", "", "Vehicles:")
        lines += formatVehicles(obj.getJSONArray("vehicles").toString())
        return lines
    }

    fun formatRouteStops(json: String): List<String> {
        val obj = JSONObject(json)
        val route = obj.getJSONObject("route")
        val stops = obj.getJSONArray("stops")
        val lines = mutableListOf<String>()
        lines += "Route: ${route.optString("ShortName")} - ${route.optString("LongName")}"
        lines += ""
        for (i in 0 until stops.length()) {
            val stop = stops.getJSONObject(i)
            lines += "${i + 1}. ${stop.optInt("StopId")} | ${stop.optString("Name")}"
        }
        return lines
    }

    fun formatDepartures(json: String): List<String> {
        val obj = JSONObject(json)
        val board = obj.getJSONObject("board")
        val enrichedIndex = mutableMapOf<String, JSONObject>()
        val enriched = obj.optJSONArray("enriched_groups") ?: JSONArray()
        for (i in 0 until enriched.length()) {
            val item = enriched.getJSONObject(i)
            enrichedIndex[item.optString("route_and_direction")] = item
        }

        val lines = mutableListOf<String>()
        lines += "Departures At Stop: ${board.optString("stop_name")}"
        val updated = board.optString("last_updated")
        if (updated.isNotBlank()) {
            lines += "Last Updated: $updated"
        }
        lines += ""

        val groups = board.optJSONArray("groups") ?: JSONArray()
        if (groups.length() == 0) {
            return listOf("No upcoming departures listed for this stop")
        }

        for (i in 0 until groups.length()) {
            val group = groups.getJSONObject(i)
            val key = group.optString("route_and_direction")
            lines += "Service: $key"
            val times = group.optJSONArray("times") ?: JSONArray()
            if (times.length() == 0) {
                lines += "Upcoming departures: none listed"
            } else {
                lines += "Upcoming departures at this stop: ${jsonArrayToList(times).joinToString(", ")}"
            }
            enrichedIndex[key]?.let { entry ->
                val routeShort = entry.optString("matched_route_short_name")
                val routeLong = entry.optString("matched_route_long_name")
                val routeId = entry.optInt("matched_route_id")
                if (routeId != 0) {
                    lines += "Matched route: $routeShort - $routeLong (RouteId $routeId)"
                }
                appendVehicles(lines, "Buses likely approaching this stop on this service:", entry.optJSONArray("live_vehicles"))
                if (entry.optJSONArray("live_vehicles")?.length() == 0) {
                    appendVehicles(lines, "Live buses on this service direction:", entry.optJSONArray("direction_vehicles"))
                }
                if ((entry.optJSONArray("live_vehicles")?.length() == 0) && (entry.optJSONArray("direction_vehicles")?.length() == 0)) {
                    appendVehicles(lines, "Live buses on the matched route:", entry.optJSONArray("route_vehicles"))
                }
            }
            lines += ""
        }
        return lines
    }

    private fun appendVehicles(lines: MutableList<String>, title: String, vehicles: JSONArray?) {
        if (vehicles == null || vehicles.length() == 0) return
        lines += title
        for (i in 0 until vehicles.length()) {
            val vehicle = vehicles.getJSONObject(i)
            val label = vehicle.optString("name").ifBlank { vehicle.optInt("vehicle_id").toString() }
            val direction = vehicle.optString("direction_long").ifBlank { vehicle.optString("direction") }
            val distance = vehicle.optDouble("distance_miles")
            val stopAway = vehicle.optInt("stops_away", -1)
            val approximate = vehicle.optBoolean("approximate_stops_away", false)
            val summary = buildString {
                append("Bus $label")
                if (stopAway >= 0) {
                    if (stopAway == 0) {
                        if (approximate) append(" | approximately at this stop") else append(" | at this stop")
                    } else {
                        if (approximate) append(" | approximately $stopAway stops away") else append(" | $stopAway stops away")
                    }
                }
                if (direction.isNotBlank()) append(" | $direction")
                if (distance > 0) append(" | ${"%.1f".format(distance)} mi away")
            }
            lines += summary
            val current = vehicle.optString("current_stop")
            val last = vehicle.optString("last_stop")
            if (current.isNotBlank()) {
                lines += "Current stop: $current"
            } else if (last.isNotBlank()) {
                lines += "Last stop: $last"
            }
            val deviation = vehicle.optInt("deviation")
            val status = vehicle.optString("display_status")
            lines += if (deviation != 0) "Status: $status ($deviation min)" else "Status: $status"
        }
    }

    private fun jsonArrayToList(array: JSONArray): List<String> {
        val values = mutableListOf<String>()
        for (i in 0 until array.length()) {
            values += array.optString(i)
        }
        return values
    }
}
