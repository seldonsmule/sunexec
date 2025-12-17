package main

/* simple get sunrise/set info */

import (
        "fmt"
        "os/exec"
        "time"
        "os"
        "bytes"
        "flag"
        "io/ioutil"
        "encoding/json"
        "encoding/base64"
        "github.com/seldonsmule/logmsg"
        "github.com/seldonsmule/tempest"

)

type RiseSet struct {

  LocationName string `json:"location_name"`

  Sunrise int `json:"sunrise"`
  Sunset int `json:"sunset"`

}

type WeatherConf struct { // info for using the developer.here.com API

  confFile string
  dataFound bool

  WeatherFlowUrl string
  WeatherFlowStationId string
  WeatherFlowToken string

}

func NewWeather() *WeatherConf{

  w := new(WeatherConf)

  w.confFile = fmt.Sprintf("%s/tmp/.sunexec.json", os.Getenv("HOME"))

  w.dataFound = false

  return w

}

func (pW *WeatherConf) SetWeatherFlow(stationid string, token string) {

  pW.WeatherFlowStationId = stationid
  pW.WeatherFlowToken = token
  pW.WeatherFlowUrl = "https://swd.weatherflow.com/swd/rest/better_forecast?station_id=" + stationid + "&token=" + token

  pW.dataFound = true
}

func (pW *WeatherConf) SetWeather(apikey string, zipcode string) {

  pW.dataFound = true
}


func (pW *WeatherConf) ReadConf() bool{

  pW.dataFound = false

  file, err := os.Open(pW.confFile) // for read access

  if (err != nil){
    logmsg.Print(logmsg.Error,"Unable to open configfile: ", err," ", pW.confFile)

     //fmt.Println("Unable to read confifile: ", err, " ", pW.confFile)
    return false
  }

  defer file.Close()

  data := make([]byte, 1000)

  count, err := file.Read(data)

  if err != nil {
     logmsg.Print(logmsg.Error,"Unable to read config: ", err, count)
     return false
  }

  err = json.NewDecoder(bytes.NewReader(data)).Decode(pW)

  if err != nil {
     logmsg.Print(logmsg.Error,"Unable to decode config: ", err)
     return false
  }


  pW.dataFound = true

  return true
}

func (pW *WeatherConf) SaveConf() bool{

  j, err := json.Marshal(pW)

  if(err != nil){
    fmt.Println(err)
    return false
  }

  fmt.Println("Saving config: ", pW.confFile)

  writeFile, err := os.Create(pW.confFile)

  if err != nil {
     logmsg.Print(logmsg.Error,"Unable to write config: ", err)
     fmt.Println("Unable to write config: ", err)
     return false
  }

  defer writeFile.Close()

  writeFile.Write(j)
  writeFile.Close()

  return true

}

func (pW *WeatherConf) Dump(){

  fmt.Println("Dumping Weather")

  fmt.Println("WeatherFlowStationId: ", pW.WeatherFlowStationId)
  fmt.Println("WeatherFlowToken: ", pW.WeatherFlowToken)
  fmt.Println("WeatherFlowUrl: ", pW.WeatherFlowUrl)


}


func testSunFile(extension string) bool {

  lockfile := fmt.Sprintf("%s/tmp/sunexec.%s", os.Getenv("HOME"), extension)

  _, statErr := os.Stat(lockfile)

  if(os.IsNotExist(statErr)){
    return false
  }

  return true;

}

func deleteSunFile(extension string){

  lockfile := fmt.Sprintf("%s/tmp/sunexec.%s", os.Getenv("HOME"), extension)

  _, statErr := os.Stat(lockfile)

  // if lock file already exist - just log it and exit
  if(statErr == nil){

    //fmt.Println("deleteSunFile ", lockfile, " created: ",info.ModTime())
    os.Remove(lockfile);

    return;

  }else{
    fmt.Printf("deleteSunFile[%s] already deleted\n", lockfile);
  }

}

func createSunFile(extension string){

  lockfile := fmt.Sprintf("%s/tmp/sunexec.%s", os.Getenv("HOME"), extension)

  info, statErr := os.Stat(lockfile)

  // if lock file already exist - just log it and exit
  if(statErr == nil){

    fmt.Println("creatSunFile ", lockfile, " created: ",info.ModTime())

    return;

  }

  // otherwise create it

  lockWriteFile, openErr := os.Create(lockfile)

  if(openErr != nil){

    fmt.Println("Error creating SunFile: ", lockfile );

    return;

  }

  fmt.Println("Created SunFile: ", lockfile );

  lockWriteFile.Close()


}


func testLockfile() bool {

  return(testSunFile("lck"))

}

func deleteLockfile(){

  deleteSunFile("lck")

}

func createLockfile(){

  createSunFile("lck")

}


func getWeatherFlowSunTimes(pW *WeatherConf) (time.Time, time.Time) {

  //var forecast BetterForcast
  var risesettimes RiseSet

  temp := tempest.New(pW.WeatherFlowToken, pW.WeatherFlowStationId)
  //temp.HelloWorld()

  timeRise := time.Now()
  timeSet := time.Now()

  jsonfile := fmt.Sprintf("%s/tmp/weatherflow.json", os.Getenv("HOME"))

  info, statErr := os.Stat(jsonfile)

  // logic so we don't call the API to much 
  // we call it once a day

  if(statErr == nil){

    logmsg.Print(logmsg.Info, "Last ", jsonfile, " write - ",info.ModTime())

    today := time.Now()

    if(today.Day() != info.ModTime().Day()){
      logmsg.Print(logmsg.Info, "not today - delete sun.json")
      os.Remove(jsonfile)
    }

  }

  jsonReadFile, openErr := os.Open(jsonfile)
  

  if(openErr == nil){

    logmsg.Print(logmsg.Debug01, "found the json file of weatherinfo");
    fmt.Println("jsonReadFile: ", jsonfile)

    byteValue, _ := ioutil.ReadAll(jsonReadFile)

    json.Unmarshal([]byte(byteValue), &risesettimes)

    jsonReadFile.Close()

    logmsg.Print(logmsg.Debug02, "risesettimes.LocationName: ", risesettimes.LocationName)

  }else{

    logmsg.Print(logmsg.Info,"Need new cache file: ", jsonfile)

    worked, forecast := temp.GetBetterForecast()  

    if(!worked){
      logmsg.Print(logmsg.Error,"Call to Weatherflow failed")
      return timeRise, timeSet
    }

    //fmt.Println("forecast: ", forecast)

    fmt.Println("locationName: ", forecast.LocationName)

    risesettimes.LocationName = forecast.LocationName
    risesettimes.Sunrise = forecast.Forecast.Daily[0].Sunrise
    risesettimes.Sunset = forecast.Forecast.Daily[0].Sunset

    jsonData, _ := json.Marshal(risesettimes)

    //fmt.Println(string(jsonData))

    jsonWriteFile, err := os.Create(jsonfile)

    if err != nil {
       panic(err)
    }
    defer jsonWriteFile.Close()

    jsonWriteFile.Write(jsonData)
    jsonWriteFile.Close()

  } //end else

 
  timeRise = time.Unix(int64(risesettimes.Sunrise), 0)
  timeSet = time.Unix(int64(risesettimes.Sunset), 0)


  return timeRise, timeSet
}


func help(){

  fmt.Println("Commands and options:\n")
  fmt.Println("-cmd setweatherflow -weatherstationid stationid -weatherstationtoken token")
  fmt.Println("\tBuilds the weather conf file.  You need the station id and developer token from your WeatherFlow system.  Got to setup to find it.")
  fmt.Println("-cmd show");
  fmt.Println("\tShow contents of config files")
  fmt.Println("-cmd printtimes");
  fmt.Println("\tDisplays sunrise/sunset times")
  fmt.Println("-cmd lock");
  fmt.Println("\tCreates lockfile to disable time logic");
  fmt.Println("-cmd unlock");
  fmt.Println("\tDeletes lockfile to renable time logic");
  fmt.Println("-cmd testlock");
  fmt.Println("\tTest and reports if a lockfile is in place");
  fmt.Println("-cmd help");
  fmt.Println("\tDisplay this help message");
  

}

func buildAuthToken(authString string){

  encodeFileName := fmt.Sprintf("%s/tmp/.sun.token", os.Getenv("HOME"))

  fmt.Println(authString)
  sEnc := base64.StdEncoding.EncodeToString([]byte(authString))
  fmt.Println(sEnc)

  encodeWriteFile, err := os.Create(encodeFileName)

  if err != nil {
     panic(err)
  }
  defer encodeWriteFile.Close()

  encodeWriteFile.Write([]byte(sEnc))
  encodeWriteFile.Close()


}

func printWeatherFlowTimes(pW *WeatherConf){

  timeRise, timeSet := getWeatherFlowSunTimes(pW)

  fmt.Println("Sunrise: ", timeRise)
  fmt.Println("Sunset: ", timeSet)
}

func execcmd(comment string, cmd string){

    msg := fmt.Sprintf("Exec %s Cmd[%s]", comment, cmd)
    logmsg.Print(logmsg.Info, msg)
    fmt.Println(msg)

    //mycmd := exec.Command(cod)
    mycmd := exec.Command("/bin/zsh", cmd)

    err := mycmd.Run()

    if(err != nil){
      logmsg.Print(logmsg.Error, "Error executing command")
      logmsg.Print(logmsg.Error, err)
    }

}

func execDayNight(sunrisecmd string, sunsetcmd string, forcesunset bool, pW *WeatherConf) {

  if(testLockfile()){

    fmt.Println("Stopping logic - lockfile exist - use unlock to remove");
    logmsg.Print(logmsg.Info, "Stopping logic - lockfile exist - use unlock to remove");

    return

  }

  if(sunrisecmd == "notset"){
    logmsg.Print(logmsg.Info, "skipping sunrise logic - cmd not set")
  }

  now := time.Now()

  timeRise, timeSet := getWeatherFlowSunTimes(pW)

  logmsg.Print(logmsg.Info, "Sunrise: ", timeRise)
  logmsg.Print(logmsg.Info, "Sunset: ", timeSet)

  //time.Sleep(time.Until(timeSet))

  // test for after sunrise

  if(forcesunset){
    fmt.Println("TEST MODE - faking that it is now night time")
  }

  if(now.After(timeRise) && now.Before(timeSet) && !forcesunset){

    logmsg.Print(logmsg.Info, "Sun has rose")

    if(testSunFile("rose")){
      logmsg.Print(logmsg.Info, "Already seen - Skipping SunRise exec")
      fmt.Println("Already seen - Skipping SunRise exec")
      return
    }

    logmsg.Print(logmsg.Info, "1st time we noticed the sun is up");
    fmt.Println("1st time we noticed the sun is up");
    createSunFile("rose");
    logmsg.Print(logmsg.Info, "Removing Sunset lock marker");
    fmt.Println("Removing Sunset lock marker");
    deleteSunFile("set");

    // if here - running our command!

    execcmd("Sunrise", sunrisecmd)

    return

  }else{

    logmsg.Print(logmsg.Info, "Sun has set")

    if(testSunFile("set")){
      logmsg.Print(logmsg.Info, "Already seen - Skipping SunSet exec")
      fmt.Println("Already seen - Skipping SunSet exec")
      return
    }

    logmsg.Print(logmsg.Info, "1st time we noticed the sun is down");
    fmt.Println("1st time we noticed the sun is down");
    createSunFile("set");
    logmsg.Print(logmsg.Info, "Removing Sunrose lock marker");
    fmt.Println("Removing Sunrose lock marker");
    deleteSunFile("rose");

    // if here - running our command!

    execcmd("Sunset", sunsetcmd)

    return

  }


}


func main() {

  fmt.Println("Test for sunrise and sunset times and then execute passed in commands\n")

  logfile := fmt.Sprintf("%s/tmp/sunexec.log", os.Getenv("HOME"))
  configfile := fmt.Sprintf("%s/tmp/.sunexec.conf", os.Getenv("HOME"))

  cmdPtr := flag.String("cmd", "help", "Command to run")
  confPtr := flag.String("conffile", configfile, "path and name of configfile")

  sunrisecmdPtr := flag.String("sunrisecmd", "notset", "cmd to execute at sunrise")
  sunsetcmdPtr := flag.String("sunsetcmd", "notset", "cmd to execute at sunset")

  forcesunsetPtr := flag.Bool("forcesunset", false, "allows us to fake out and test in the daytime")


  weatherStationIdPtr := flag.String("weatherstationid", "notset", "Weather station id to use for weather (see weatherflow app for info")
  weatherStationTokenPtr := flag.String("weatherstationtoken", "notset", "Weather station token to use for weather (see weatherflow app for info")

  cameraPtr := flag.Int("camera", 0, "SecuritySpy camera number")
  presetPtr := flag.Int("preset", 0, "SecuritySpy preset number")

  daypresetPtr := flag.Int("daypreset", 2, "SecuritySpy preset number for day")
  nightpresetPtr := flag.Int("nightpreset", 1, "SecuritySpy preset number for night")

  WConf := NewWeather()
  if(!WConf.ReadConf()){
    fmt.Println("Error reading config file - initalizing instead\n\n")
    WConf.SaveConf()
    if(!WConf.ReadConf()){
      fmt.Println("Error reading config file after initalizing - something is wrong\n\n")
      os.Exit(1)
    }
  }
  /*
  if(!WConf.ReadConf()){
    fmt.Println("Error reading config file\n\n")
    os.Exit(1)
  }
  */

  flag.Parse()

  if(*cmdPtr == "help"){
    help()
    os.Exit(1)
  }


  logmsg.SetLogLevel(logmsg.Debug03)

  logmsg.SetLogFile(logfile)

  logmsg.Print(logmsg.Info, "cmdPtr = ", *cmdPtr)
  logmsg.Print(logmsg.Info, "confPtr = ", *confPtr)
  logmsg.Print(logmsg.Info, "weatherStationIdPtr = ", *weatherStationIdPtr)
  logmsg.Print(logmsg.Info, "weatherStationTokenPtr = ", *weatherStationTokenPtr)
  logmsg.Print(logmsg.Info, "cameraPtr = ", *cameraPtr)
  logmsg.Print(logmsg.Info, "presetPtr = ", *presetPtr)
  logmsg.Print(logmsg.Info, "daypresetPtr = ", *daypresetPtr)
  logmsg.Print(logmsg.Info, "nightpresetPtr = ", *nightpresetPtr)
  logmsg.Print(logmsg.Info, "sunrisecmdPtr = ", *sunrisecmdPtr)
  logmsg.Print(logmsg.Info, "sunsetcmdPtr = ", *sunsetcmdPtr)
  logmsg.Print(logmsg.Info, "forcesunsetPtr = ", *forcesunsetPtr)
  logmsg.Print(logmsg.Info, "tail = ", flag.Args())

  switch *cmdPtr {

    case "show":
      if(WConf.dataFound){
        WConf.Dump()
      }else{
        fmt.Println("ERROR-> Weather Conf data not found")
      }
/*
      printTimes(WConf)
*/

    case "printtimes":
      fallthrough
    case "printweatherflowtimes":
      printWeatherFlowTimes(WConf)

    case "setweatherflow":
      if(*weatherStationIdPtr == "notset"){
        fmt.Println("Err:  Missing weatherstationid paramater")
        os.Exit(2)
      }

      if(*weatherStationTokenPtr == "notset"){
        fmt.Println("Err:  Missing weatherstationtoken paramater")
        os.Exit(2)
      }

      WConf.SetWeatherFlow(*weatherStationIdPtr, *weatherStationTokenPtr)
      WConf.Dump()
      if(!WConf.SaveConf()){
        fmt.Println("Err: Unable to save weather file")
        os.Exit(2)
      }

    case "testlock":
      if(testLockfile()){
        fmt.Println("Lockfile found - restricken in place")
      }else{
        fmt.Println("Lockfile NOT found - game on")
      }

    case "reset":
      fmt.Println("Removing lockfile and surise/set marker files")
      deleteSunFile("lck")
      deleteSunFile("rose")
      deleteSunFile("set")
 
    case "exec":

      if(!WConf.dataFound){
        fmt.Println("Err: Missing Weather Conf data - Use -cmd setweatherflow")
        os.Exit(2)
      }

      fmt.Println("Checking to see if it is time to run something")
      execDayNight(*sunrisecmdPtr, *sunsetcmdPtr, *forcesunsetPtr, WConf)

    case "lock":
      fmt.Println("Locking time logic - use unlock to restore");
      createLockfile();

    case "unlock":
      fmt.Println("Removing lock time logic - going back to normal operations");
      deleteLockfile();

    case "help":
      help()
      os.Exit(0)

    default:
      help()
      os.Exit(2)

  } // end switch


}
