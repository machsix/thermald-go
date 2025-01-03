#!/bin/bash

# Function to check if a drive exists and retrieve its temperature
get_drive_temperature() {
  local drive="$1"
  local info="$(sudo smartctl -a $drive)"
  echo "${info}" | awk '
BEGIN {
  temp=0
  printf "{\n  \"type\": \"hdd\",\n"
}
/Serial Number:/ {
  split($0, i, ":")
  gsub(/ /, "", i[2])
  printf "  \"id\": \"%s\",\n", i[2]
}
/Device Model/ {
  split($0, i, ":")
  gsub(/ /, "", i[2])
  printf "  \"model\": \"%s\",\n", i[2]
}
/194 Temp/ {
  temp=$10
  exit
}
/190 Airflow/ {
  temp=$10
  exit
}
/Temperature Sensor 1:/ {
  temp=$10
  exit
}
/Current Drive Temperature:/ {
  temp=$4
  exit
}
/Temperature:/ {
  temp=$4
  exit
}
END {
  printf "  \"temperature\": %.1f\n}", temp
}
'
}

# Function to retrieve the core temperature
get_core_temperature() {
  local zone=$1
  local file="/sys/class/thermal/${zone}/temp"
  temperature=$(cat ${file} | sed 's/\(.\)..$/.\1/')
  printf "{\n  \"type\": \"cpu\",\n  \"id\": \"${zone}\",\n  \"temperature\": %.1f\n}" $temperature
}

declare -a data=()
for i in /sys/class/thermal/thermal_zone*/temp; do
  zone=$(echo $i | cut -d'/' -f5)
  data+=("$(get_core_temperature $zone)")
  # echo $info
done

# Print drive temperatures
for drive in /dev/sd?; do
  if [ -e "$drive" ]; then
    data+=("$(get_drive_temperature "$drive")")
  fi
done

printf "["
for i in "${!data[@]}"; do
  if [ "$i" -lt "$(( ${#data[@]} - 1 ))" ]; then
    printf '%s,\n' "${data[i]}"
  else
    printf '%s]\n' "${data[i]}"
  fi
done
