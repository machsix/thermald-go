#!/bin/bash

# Function to check if a drive exists and retrieve its temperature
get_drive_temperature() {
  local drive="$1"
  local info="$(sudo smartctl -a $drive)"
  local temp=$(echo "$info" | grep '194 Temp' | awk '{print $10}')
  if [[ $temp == '' ]]; then
    temp=$(echo "$info" | grep '190 Airflow' | awk '{print $10}')
  fi
  if [[ $temp == '' ]]; then
    temp=$(echo "$info" | grep 'Temperature Sensor 1:' | awk '{print $4}')
  fi
  if [[ $temp == '' ]]; then
    temp=$(echo "$info" | grep 'Current Drive Temperature:' | awk '{print $4}')
  fi
  if [[ $temp == '' ]]; then
    temp=$(echo "$info" | grep 'Temperature:' | awk '{print $2}')
  fi
  echo "$temp"
}

# Function to retrieve the core temperature
get_core_temperature() {
  # local core_temp=$(sensors | grep 'Core 0:' | awk '{print $3}')
  # echo "$core_temp"
  for zone in /sys/class/thermal/thermal_zone*/temp; do
    temperature=$(cat $zone | sed 's/\(.\)..$/.\1°C/')
    #zone_name=$(echo $zone | cut -d'/' -f5)
    zone_name=$(echo $zone | sed 's/.*_\(zone.*\)\/.*/cpu_\1/')
    echo "${zone_name}: ${temperature}"
  done
  #paste <(cat /sys/class/thermal/thermal_zone*/type) <(cat /sys/class/thermal/thermal_zone*/temp) | column -s $'\t' -t | sed 's/\(.\)..$/.\1°C/'
}

# Get and print core temperature first
get_core_temperature

# Print drive temperatures
for drive in /dev/sd?; do
  if [ -e "$drive" ]; then
    temperature=$(get_drive_temperature "$drive")
    echo "$drive: $temperature°C"
  fi
done
