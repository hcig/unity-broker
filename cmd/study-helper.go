package main

import (
  "encoding/csv"
  "encoding/json"
  "fmt"
  "os"
  "time"
  "viveSyncBroker/pb/broker/messages"
)

var EntityProperties = []string{"x", "y", "rx", "ry", "rz"}

type Set[T any] struct {
  m map[T]bool
}

func (s *Set[T]) Add(val T) bool {
  _, found := (*s).m[val]
  if found {
    return false
  }
  (*s).m[val] = true
  return true
}
func (s *Set[T]) Slice() []T {
  result := make([]T, 0, len(s.m))
  for elem, _ := range s.m {
    result = append(result, elem)
  }
  return result
}

func main() {

}

func getFirstTimestamp(fileName string) (*time.Time, error) {
  data, err := openCsvFile(fileName)
  if err != nil {
    return nil, err
  }
  var row []string
  for {
    row, err = data.Read()
    if err != nil {
      if err == csv.ErrFieldCount {
        break
      }
      t, err := time.Parse(time.RFC3339Nano, row[0])
      if err == nil && !t.IsZero() {
        return &t, nil
      }
    }
  }
  return nil, fmt.Errorf("could not find first timestamp in csv")
}

func openCsvFile(fileName string) (*csv.Reader, error) {
  data, err := os.Open(fileName)
  if err != nil {
    return nil, err
  }
  return csv.NewReader(data), nil
}

func extractCvsData(fileName string, outFileName string) error {
  data, err := openCsvFile(fileName)
  if err != nil {
    return err
  }
  resultFile, err := os.OpenFile(outFileName, os.O_CREATE | os.O_RDWR, 0777)
  if err != nil {
    return err
  }
  resultCsv := csv.NewWriter(resultFile)
  entities := Set[string]{}
  startTimeStamp, err := getFirstTimestamp(fileName)
  if err != nil {
    return err
  }
  current := map[string]interface{}{}
  csvFull, err := data.ReadAll()
  if err != nil {
    return err
  }
  for j, rowRaw := range csvFull {
    row := messages.Command{}
    if err = json.Unmarshal([]byte(rowRaw[1]), &row); err != nil {
      return err
    }
    if row.Command == messages.CommandType_UpdateCommand && row.Payload.GetUpdate().Name != "" {
      if row.Payload.GetUpdate().GetTrigger() != nil {
        // No position data, just active
        entities.Add(row.Payload.GetUpdate().Name)
      } else {
        for _, prop := range EntityProperties {
          entities.Add(row.Payload.GetUpdate().Name + "_" + prop)
        }
      }
    }
    curTime := row.Timestamp
    nextTime := curTime
    const csvHeader = []string{'Timestamp', 'Eyes_x', 'Eyes_y', ...entities}
    resultCsv.Write(append(csvHeader, entities.Slice()...))

// 2nd run: Collect actual data
for (let j = 0; j < data[i].length; j++) {
// Assign new value to current object
if (!data[i][j][1].hasOwnProperty('payload')) continue;
// if (data[i][j][1].payload.name === 'Car_8_aussenring_0') {
//   console.log(data[i][j][1].payload);
// }
let pos = {};
let savePoint = false;
if (data[i][j][1].payload.hasOwnProperty('rigidbody')) {
pos = Object.assign({}, data[i][j][1].payload.rigidbody.position);
// Rigidbody does not send deltas, thus change to absolute
data[i][j][1].payload.type = 1;
savePoint = true;
} else if (data[i][j][1].payload.hasOwnProperty('transform')) {
pos = Object.assign({}, data[i][j][1].payload.transform.position);
savePoint = true;
} else if (data[i][j][1].payload.hasOwnProperty('tracker')) {
if (data[i][j][1].payload.tracker.values.direction.z < 1) {
// If z is 0, no tracking data has been transferred
continue;
}
current['Eyes_x'] = data[i][j][1].payload.tracker.values.direction.x;
current['Eyes_y'] = data[i][j][1].payload.tracker.values.direction.y;
} else if (data[i][j][1].payload.hasOwnProperty('trigger') && data[i][j][1].payload.trigger.other == = 'Player') {
//pos = Object.assign({}, data[i][j][1].payload.transform.position);
//console.error(data[i][j][1].payload);
current[data[i][j][1].payload.name] = data[i][j][1].payload.trigger.active;
} else {
//console.error("No transform", data[i][j][1].payload);
continue;
}
curTime = new Date(data[i][j][1].timestamp);
if (savePoint == = true) {
if (data[i][j][1].payload.type == = 0) {
current[data[i][j][1].payload.name + '_x'] += pos.x || 0;
current[data[i][j][1].payload.name + '_y'] += pos.z || 0;
} else {
current[data[i][j][1].payload.name + '_x'] = pos.x || 0;
current[data[i][j][1].payload.name + '_y'] = pos.z || 0;
}
// When a point changes, its distances also change
if (data[i][j][1].payload.name == = 'Camera') {
for (const diName of distances) {
if (current[diName + '_x'] != = null) {
current[diName] = distanceFields(current, diName, 'Camera');
}
}
} else {
current[data[i][j][1].payload.name] = distanceFields(current, data[i][j][1].payload.name, 'Camera');
}
}
if (data[i][j][1].payload && data[i][j][1].payload.name) {
// First entry sets Zero-Point, else every time slice
current.Timestamp = Math.abs(nextTime - startTimeStamp);
if (curTime >= nextTime) {
await resultFile.write(getValuesSortedByHeader(current, csvHeader).join(CSV_SEPARATOR) + "\n");
nextTime.setMilliseconds(nextTime.getMilliseconds() + TIME_SLICE_MS);
}
}
}
if (resultFile) await resultFile.close()
}

return nil
}
