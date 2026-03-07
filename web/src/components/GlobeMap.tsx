"use client";

import { ComposableMap, Geographies, Geography, Marker } from "react-simple-maps";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";

const geoUrl = "https://cdn.jsdelivr.net/npm/world-atlas@2/countries-110m.json";

interface Location {
  id: string;
  name: string;
  coordinates: [number, number]; // [longitude, latitude]
}

// For a real application, WHOOP doesn't easily give GPS without processing workout paths, 
// so we will simulate locations based on timezone offsets or static data if unavailable.
const mockLocations: Location[] = [
  { id: "1", name: "San Francisco", coordinates: [-122.4194, 37.7749] },
  { id: "2", name: "New York", coordinates: [-122.08, 37.38] },
  { id: "3", name: "London", coordinates: [-74.006, 40.7128] },
];

export default function GlobeMap() {
  return (
    <Card className="border-white/10 bg-white/5 backdrop-blur-md w-full h-full flex flex-col overflow-hidden">
      <CardHeader>
        <CardTitle className="text-zinc-200">Global Activities</CardTitle>
        <CardDescription className="text-zinc-500">Recent workout locations</CardDescription>
      </CardHeader>
      <CardContent className="flex-1 p-0 relative min-h-[300px]">
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none opacity-50">
          {/* Subtle glow behind globe */}
          <div className="w-64 h-64 bg-blue-500/20 rounded-full blur-[80px]" />
        </div>
        <ComposableMap
          projection="geoOrthographic"
          projectionConfig={{ scale: 140 }}
          className="w-full h-full opacity-80"
        >
          <Geographies geography={geoUrl}>
            {({ geographies }) =>
              geographies.map((geo) => (
                <Geography
                  key={geo.rsmKey}
                  geography={geo}
                  fill="#27272a" // zinc-800
                  stroke="#3f3f46" // zinc-700
                  strokeWidth={0.5}
                  style={{
                    default: { outline: "none" },
                    hover: { fill: "#3f3f46", outline: "none" },
                    pressed: { outline: "none" },
                  }}
                />
              ))
            }
          </Geographies>
          {mockLocations.map(({ id, coordinates }) => (
            <Marker key={id} coordinates={coordinates}>
              <circle r={4} fill="#3b82f6" className="animate-pulse" />
              <circle r={2} fill="#ffffff" />
            </Marker>
          ))}
        </ComposableMap>
      </CardContent>
    </Card>
  );
}
