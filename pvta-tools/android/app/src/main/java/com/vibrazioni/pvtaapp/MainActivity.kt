package com.vibrazioni.pvtaapp

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.ScrollableTabRow
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                PvtaApp()
            }
        }
    }
}

private enum class Screen(val label: String) {
    Routes("Routes"),
    Vehicles("Vehicles"),
    Stop("Find Stop"),
    Departures("Departures"),
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun PvtaApp() {
    val bridge = remember { BridgeClient() }
    val scope = rememberCoroutineScope()
    val focusManager = LocalFocusManager.current
    val listState = rememberLazyListState()
    val lines = remember {
        mutableStateListOf(
            "PVTA Tools",
            "",
            "Use the tabs above to load routes, vehicles, stop details, or departures.",
        )
    }
    var screen by remember { mutableStateOf(Screen.Departures) }
    var routeInput by remember { mutableStateOf("") }
    var stopInput by remember { mutableStateOf("") }
    var stopFilter by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var resultTitle by remember { mutableStateOf("Results") }

    fun load(title: String, block: suspend () -> List<String>) {
        scope.launch {
            focusManager.clearFocus()
            loading = true
            runCatching { block() }
                .onSuccess {
                    resultTitle = title
                    lines.clear()
                    lines.addAll(it)
                }
                .onFailure {
                    resultTitle = title
                    lines.clear()
                    lines += "Error: ${it.message}"
                }
            loading = false
            listState.scrollToItem(0)
        }
    }

    LaunchedEffect(screen) {
        listState.scrollToItem(0)
    }

    Column(modifier = Modifier.fillMaxSize()) {
        TopAppBar(title = { Text("PVTA Tools") })
        Column(modifier = Modifier.padding(16.dp)) {
            Text(
                text = "Quick access to routes, live buses, stop lookup, and stop departures.",
                style = MaterialTheme.typography.bodyMedium,
            )
            Spacer(modifier = Modifier.height(12.dp))
            ScrollableTabRow(selectedTabIndex = screen.ordinal) {
                Screen.entries.forEach { entry ->
                    Tab(
                        selected = screen == entry,
                        onClick = { screen = entry },
                        text = { Text(entry.label) },
                    )
                }
            }
            Spacer(modifier = Modifier.height(12.dp))
            when (screen) {
                Screen.Routes -> RoutesPanel(
                    routeInput = routeInput,
                    onRouteInputChange = { routeInput = it },
                    stopFilter = stopFilter,
                    onStopFilterChange = { stopFilter = it },
                    onLoadRoutes = { load("Routes") { JsonFormatters.formatRoutes(bridge.routes("")) } },
                    onLoadRoute = { load("Route Status") { JsonFormatters.formatRouteStatus(bridge.routeStatus("", routeInput)) } },
                    onLoadRouteStops = { load("Route Stops") { JsonFormatters.formatRouteStops(bridge.routeStops("", routeInput)) } },
                    onLoadStops = { load("Stops") { JsonFormatters.formatStops(bridge.stops("", stopFilter)) } },
                )
                Screen.Vehicles -> VehiclesPanel(
                    onLoadVehicles = { load("Vehicles") { JsonFormatters.formatVehicles(bridge.vehicles("")) } },
                )
                Screen.Stop -> StopPanel(
                    stopInput = stopInput,
                    onStopInputChange = { stopInput = it },
                    onLoadStopStatus = { load("Stop Details") { JsonFormatters.formatStopStatus(bridge.stopStatus("", stopInput)) } },
                )
                Screen.Departures -> DeparturesPanel(
                    stopInput = stopInput,
                    onStopInputChange = { stopInput = it },
                    onLoadDepartures = { load("Departures") { JsonFormatters.formatDepartures(bridge.departures("", stopInput)) } },
                )
            }
            Spacer(modifier = Modifier.height(12.dp))
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f),
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(12.dp),
                ) {
                    Text(resultTitle, style = MaterialTheme.typography.titleMedium)
                    Spacer(modifier = Modifier.height(4.dp))
                    if (loading) {
                        Text("Loading...")
                        Spacer(modifier = Modifier.height(8.dp))
                    }
                    LazyColumn(
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f),
                        state = listState,
                    ) {
                        items(lines) { line ->
                            Text(
                                text = line,
                                modifier = Modifier.padding(vertical = 2.dp),
                                fontFamily = FontFamily.Monospace,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun RoutesPanel(
    routeInput: String,
    onRouteInputChange: (String) -> Unit,
    stopFilter: String,
    onStopFilterChange: (String) -> Unit,
    onLoadRoutes: () -> Unit,
    onLoadRoute: () -> Unit,
    onLoadRouteStops: () -> Unit,
    onLoadStops: () -> Unit,
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text("Browse routes or drill into a specific route.", style = MaterialTheme.typography.bodyMedium)
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Button(onClick = onLoadRoutes) {
                    Text("All Routes")
                }
                TextButton(onClick = onLoadStops) {
                    Text("Nearby Stop List")
                }
            }
            OutlinedTextField(
                value = routeInput,
                onValueChange = onRouteInputChange,
                label = { Text("Route short name or RouteId") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Button(onClick = onLoadRoute) {
                    Text("Route Status")
                }
                TextButton(onClick = onLoadRouteStops) {
                    Text("Route Stops")
                }
            }
            OutlinedTextField(
                value = stopFilter,
                onValueChange = onStopFilterChange,
                label = { Text("Optional stop-name filter") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            TextButton(onClick = onLoadStops) {
                Text("List Stops")
            }
        }
    }
}

@Composable
private fun VehiclesPanel(
    onLoadVehicles: () -> Unit,
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text("See all live buses currently reported by PVTA.", style = MaterialTheme.typography.bodyMedium)
            Button(onClick = onLoadVehicles) {
                Text("Load Vehicles")
            }
        }
    }
}

@Composable
private fun StopPanel(
    stopInput: String,
    onStopInputChange: (String) -> Unit,
    onLoadStopStatus: () -> Unit,
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text("Look up a stop by StopId or by part of its name.", style = MaterialTheme.typography.bodyMedium)
            OutlinedTextField(
                value = stopInput,
                onValueChange = onStopInputChange,
                label = { Text("Stop id or stop name") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            Button(onClick = onLoadStopStatus) {
                Text("Stop Details")
            }
        }
    }
}

@Composable
private fun DeparturesPanel(
    stopInput: String,
    onStopInputChange: (String) -> Unit,
    onLoadDepartures: () -> Unit,
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text("Best mobile view: upcoming departures and likely live buses for a stop.", style = MaterialTheme.typography.bodyMedium)
            OutlinedTextField(
                value = stopInput,
                onValueChange = onStopInputChange,
                label = { Text("Stop id or stop name") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            Button(onClick = onLoadDepartures) {
                Text("Load Departures")
            }
        }
    }
}
