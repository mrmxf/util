## -----------------------------------------------------------------------------
## Clog Core Include Functions
##
## Usage: clog Inc --help
## -----------------------------------------------------------------------------
# simple test for some ZSH alternate paths
cIsZSH=$(echo $0 | grep "zsh")

# lightmode / darkmode colors sent before this file

# ------------------------------------------------------------------------------
#define a function to echo a message - color initialised as text

fEcho() {
  # 1st parameter might by a keyword to control color ERROR / WARNING / INFO then print the rest

  if [ -n cIsZSH ]; then
    local lastline=$(echo $functrace | tail -1)
    local ln=${lastline##*:}
  else
    # printf '%s\n' "${FUNCNAME[@]}" ; printf '%s\n' "${BASH_LINENO[@]}"
    local csl
    let csl=${#FUNCNAME[@]}-2
    local ln=${BASH_LINENO[$csl]}
    if [ $ln -lt 10 ]; then ln="0$ln"; fi
    if [ $ln -lt 100 ]; then ln="0$ln"; fi
  fi
  
  case "$1" in
    "ABORT")   shift; MSG="${cE}  ABORT$cT ";;
    "ERROR")   shift; MSG="${cE}  ERROR$cT ";;
    "WARNING") shift; MSG="${cW}WARNING$cT ";;
    "INFO")    shift; MSG="${cI}   INFO$cT ";;
    "OK")      shift; MSG="${cS}     OK$cT ";;
    "SUCCESS") shift; MSG="${cS}SUCCESS$cT ";;
    "TEXT")    shift; MSG="${cT}       $cT ";;
    *) MSG="${cT}       $cT " ;;
  esac

  # if the first parameter is a "_" then go up one line & delete it (overprint)
  # https://en.wikipedia.org/wiki/ANSI_escape_code - up1, col1, del(EOL)
  local up=""
  [[ "$1" == "_" ]] && up="\e[A\e[G\e[K" && shift

  printf "$up$cC$ln$cI>>$MSG$@$cX\n"
}

# --- log to /var/log/clog/clog.log --------------------------------------------
export LogPATH=/var/log/clog
export LogFILE=clog.log
fLog() {
  # get the calling script name if this function is `source`ed
  LogSRC=${0##*/}
  # ensure the logging folder exists
  sudo mkdir -p "$LogPATH"

  # remove highlight escape sequences from the message
  local LogMsg=$(echo "$2" | sed -e 's/\\e\[[^m]*m//g')
  local TimeStamp=$(printf '%(%Y-%m-%d %H:%M:%S)T' -1)
  LogMsg=$(printf "%-20s %-8s %-18s %s\n" "$TimeStamp" "$1" "$LogSRC" "$LogMsg")

  echo "$LogMsg" | sudo tee -a "$LogPATH/$LogFILE" >/dev/null
}

# ------------------------------------------------------------------------------
#define an abort function
fAbort() {

  #param $1 = message
  #find the calling function in the call stack (csl)
  #this is the (call stack length-1) minus another 1 because its zero based
  local csl
  let csl=${#FUNCNAME[@]}-2
  fEcho "ABORT" "${cE}Abort called from ${cX}${FUNCNAME[csl]}${cE} at line${cX} ${BASH_LINENO[csl]}"
  exit -1
}

# ------------------------------------------------------------------------------
# Helper functions for warnings etc
fInfo()    { fEcho INFO "$@"; }
fOk()      { fEcho OK "$@"; }
fSuccess() { fEcho OK "$@"; }
fText()    { fEcho TEXT "$@"; }
fWarn()    { fEcho WARNING "$@"; }
fWarning() { fEcho WARNING "$@"; }
fError()   { fEcho ERROR "$@"; }
fDivider() { fEcho "$cH-===--===--===--===--===--===--===--===--===--===--===--===--===--===-"; }

# ------------------------------------------------------------------------------
# display color highlighting example
fColors() {
  if [ $1 ]; then
    fError "this color is for$cE errors: \$cS $cX"
    fWarning "this color is for$cW warnings: \$cW $cX"
    fOk "this color is for$cS success : \$cS $cX"
    fText "this color is for$cT plain text : \$cT $cX"
    fInfo "this color is for$cI info: \$cI $cX"
    fText "this color is for$cF files : \$cF  $cX"
    fText "this color is for$cC command lines: \$cC  $cX"
    fText "this color is for$cU urls: \$cU $cX"
    fText "turn colors off with\$cX $cX to reset highlighting"
  fi
}

# ------------------------------------------------------------------------------
# get the line number of the calling function being executed from the stack
function fGetStack() {
  STACK=""
  local i message="${1:-""}"
  local stack_size=${#FUNCNAME[@]}
  # to avoid noise we start with 1 to skip the get_stack function
  for ((i = 1; i < $stack_size; i++)); do
    local func="${FUNCNAME[$i]}"
    [ x$func = x ] && func=MAIN
    local linen="${BASH_LINENO[$((i - 1))]}"
    local src="${BASH_SOURCE[$i]}"
    [ x"$src" = x ] && src=non_file_source

    STACK+=$'\n'"   at: "$func" "$src" "$linen
  done
  STACK="${message}${STACK}"
}

# ------------------------------------------------------------------------------
# Download from S3 to local
#   $1 - common prefix of the bucket
#   $2 - common options for every sync/copy
#   $3 - common destination file path
#   $4 - any string to perform dryrun
#
#   $DOWNLOAD - an array of commands e.g.
#
#   DOWNLOAD=()
#   DOWNLOAD+=("sync /                      / ")
#   DOWNLOAD+=("cp   public/favicon.ico    favicon.ico ")
#   fDownloadS3  s3://mmh-cache/bot-bdh/staging/hugo-metarex-media\
#                '--include="*"'\
#                /var/www/mySite\
#                DryRun

fCpS3() {
  # iterate through SYNCS - print & execute
  local PREFIX=$1
  local OPTION=$2
  local FOLDER=$3

  SRC=()
  DST=()
  VRB=()
  SRCMAX=0
  DSTMAX=0

  for d in "${!DOWNLOAD[@]}"; do
    TOK=(${DOWNLOAD[$d]})    #use bash built-in tokenisation
    VRB+=(${TOK[0]})         # append to verb array
    SRC+=($PREFIX/${TOK[1]}) # append to full source path array
    DST+=($FOLDER/${TOK[2]}) # append to Destination Folder array

    if [[ ${#SRC[$d]} -gt $SRCMAX ]]; then SRCMAX=${#SRC[$d]}; fi
    if [[ ${#DST[$d]} -gt $DSTMAX ]]; then DSTMAX=${#DST[$d]}; fi
  done
  echo "$SRCMAX---$DSTMAX"
  for d in "${!DOWNLOAD[@]}"; do
    VVV=$(printf "%-5s" "${VRB[$d]}")
    SSS=$(printf "%-${SRCMAX}s" "${SRC[$d]}")
    DDD=$(printf "%-${DSTMAX}s" "${DST[$d]}")
    printf "${cC}aws s3$cW $VVV$cC $OPTION$cU $SSS$cF $DDD$cX\n"

    #Dry run if the 4th parameter is set
    if [[ -z "$4" ]]; then
      aws s3 ${VRB[$d]} $OPTION ${SRC[$d]} ${DST[$d]}
    fi
  done
}

# ------------------------------------------------------------------------------
#     fMachine
#       sets cOS to linux | mac | windows | gitpod | unknwown
#       sets cPU to i32   | i64 | a64                      - intel / arm
fMachine() {

  # detect what sort of linux shell we're running in
  case "$(uname -s)" in
  Linux*)
    cOS="linux"
    if ! [ -z "${GITPOD_GIT_USER_NAME+x}" ]; then
      cOS="gitpod"
    fi
    ;;
  Darwin*) cOS="mac" ;; # do this before checking for windows
  CYGWIN*) cOS="linux" ;;
  MINGW*) cOS="linux" ;;
  MSYS_NT*) cOS="linux" ;;
  win*) . cOS="windows" ;;
  *) cOS="unknown" ;;
  esac

  # detect what sort of architecturewe're running in
  case "$(uname -m)" in
  x86_64*) cPU="i64" ;;
  amd*) cPU="i64" ;;
  i686*) cPU="i32" ;;
  arm64) cPU="a64" ;;
  aarch64*) cPU="a64" ;;
  i386*) cPU="i32" ;;
  i486*) cPU="i32" ;;
  i586*) cPU="i32" ;;
  *) cPU="i32" ;;
  esac
  export cOS
  export cPU
  #echo "running on $MACHINE with $CPU architecture"
}

# ------------------------------------------------------------------------------
# make some aliases for all the functions (backwards compatibility)
#printing
fnAbort() { fAbort "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; }     # abort & print
fnError() { fError "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; }     # print ERROR xxx
fnDivider() { fDivider "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; } # print Divider
fnEcho() { fEcho "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; }       # print with line num
fnColors() { fColors "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; }   # print color test
fnInfo() { fInfo "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; }       # print INFO xxx
fnOk() { fOk "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; }           # print OK xxx
fnText() { fText "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; }       # print plain Text
fnSuccess() { fSuccess "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; } # print SUCCESS Text
fnUsage() { fUsage "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; }     # print Usage message
fnWarning() { fWarning "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; } # print WARNING xxx
#data
fnGetStack() { fGetStack "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; } # get source line number
fnMachine() { fMachine "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; }   # return cOS & cPU
#Utils
fnWget() { fnWget "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"; } # check for wget first
