package com.vibrazioni.pvtaapp

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

class BridgeClient {
    suspend fun routes(baseUrl: String): String = invokeBridge(baseUrl, "RoutesJSON")

    suspend fun vehicles(baseUrl: String): String = invokeBridge(baseUrl, "VehiclesJSON")

    suspend fun stops(baseUrl: String, filter: String): String = invokeBridge(baseUrl, "StopsJSON", filter)

    suspend fun routeStatus(baseUrl: String, input: String): String = invokeBridge(baseUrl, "RouteStatusJSON", input)

    suspend fun stopStatus(baseUrl: String, input: String): String = invokeBridge(baseUrl, "StopStatusJSON", input)

    suspend fun departures(baseUrl: String, input: String): String = invokeBridge(baseUrl, "DeparturesJSON", input)

    suspend fun routeStops(baseUrl: String, input: String): String = invokeBridge(baseUrl, "RouteStopsJSON", input)

    private suspend fun invokeBridge(baseUrl: String, methodName: String, vararg args: String): String = withContext(Dispatchers.IO) {
        val bridge = createBridge()
        if (baseUrl.isNotBlank()) {
            invokeAny(bridge, listOf("SetBaseURL", "setBaseURL"), arrayOf(baseUrl))
        }
        val result = invokeAny(bridge, listOf(methodName, methodName.replaceFirstChar { it.lowercase() }), args)
        result?.toString() ?: error("bridge returned no data for $methodName")
    }

    private fun createBridge(): Any {
        val packageFactories = listOf(
            "com.vibrazioni.pvta.bridge.pvtagoandroid.Pvtagoandroid",
            "com.vibrazioni.pvta.bridge.Mobilebridge",
            "com.vibrazioni.pvta.bridge.Pvtaandroidbridge",
            "com.vibrazioni.pvta.bridge.Mobilebridgeandroid",
        )
        for (factoryName in packageFactories) {
            runCatching {
                val clazz = Class.forName(factoryName)
                val method = clazz.methods.firstOrNull {
                    it.name.equals("newBridge", ignoreCase = true) && it.parameterCount == 0
                }
                if (method != null) {
                    return method.invoke(null) ?: error("newBridge returned null")
                }
            }
        }

        val directTypes = listOf(
            "com.vibrazioni.pvta.bridge.pvtagoandroid.Bridge",
            "com.vibrazioni.pvta.bridge.Bridge",
        )
        for (typeName in directTypes) {
            runCatching {
                val clazz = Class.forName(typeName)
                return clazz.getDeclaredConstructor().newInstance()
            }
        }

        error(
            "Go bridge class not found. Rebuild android/app/libs/pvta-mobilebridge.aar and reinstall the app.",
        )
    }

    private fun invokeAny(target: Any, names: List<String>, args: Array<out String>): Any? {
        val methods = target.javaClass.methods
        for (name in names) {
            val match = methods.firstOrNull {
                it.name.equals(name, ignoreCase = true) && it.parameterCount == args.size
            } ?: continue
            return match.invoke(target, *args)
        }
        error("method not found on bridge: ${names.first()}")
    }
}
