# Elastos.ELA

## Summary

Elacoin is the digital currency solution within Elastos eco system.

## Build

- put it under $GOPATH
- run `glide update && glide install` to install depandencies.
- then run `make` to build files.

## Run

- run ./node to run the node program.


## Mac OS 10.13 high Sierra build

Step 1. Install XCode and command line tools.
  Install xCode from Apple App Store and run for first time and close out after it finishes setting up.
  Open up Terminal and enter:
    xcode-select --install
  
Step 2. Install homebrew.
  In Terminal enter:
    /usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"
  
  After Homebrew has finished installation lets find any errors and follow the instructions to correct them, type:
    brew doctor

  If any warning occur after brew doctor, follow the instructions presented by brew to fix.
  
Step 3. Install required packages.
  In Terminal enter:
    brew install glide go pkg-config zmq
  
Step 4. Go to folder in terminal “/usr/local/Cellar/go/1.10/libexec/src”
  In Terminal enter:
    cd /usr/local/Cellar/go/1.10/libexec/src

Step 5. Clone Elastos.ELA.
  In Terminal enter:
    git clone https://github.com/elastos/Elastos.ELA.git
  
Step 6. Clone golang/crypto.
  In Terminal enter:
    cd /usr/local/Cellar/go/1.10/libexec/src/vendor
    mkdir github.com
    cd github.com
    mkdir golang
    cd golang
    git clone https://github.com/golang/crypto.git)
  
Step 7. Clone gorilla/websocket.
  In Terminal enter:
    cd /usr/local/Cellar/go/1.10/libexec/src/vendor/github.com
    mkdir gorilla
    cd gorilla
    git clone https://github.com/gorilla/websocket.git)
    
Step 8. Clone itchyny/base58-go.
  In Terminal enter:
    cd  /usr/local/Cellar/go/1.10/libexec/src/vendor/github.com/
    mkdir itchyny
    cd itchyny
    git clone https://github.com/itchyny/base58-go.git)
    
Step 9. Clone pborman/uuid.
  In Terminal enter:
    cd /usr/local/Cellar/go/1.10/libexec/src/vendor/github.com/
    mkdir pborman
    cd pborman,
    git clone https://github.com/pborman/uuid.git
    
Step 10. Clone pebbe/zmq4.
  In Terminal enter:
    cd /usr/local/Cellar/go/1.10/libexec/src/vendor/github.com/
    mkdir pebbe
    cd pebbe
    git clone https://github.com/pebbe/zmq4.git
    
Step 11. Clone syndtr/goleveldb.
  In Terminal enter:
    cd /usr/local/Cellar/go/1.10/libexec/src/vendor/github.com/
    mkdir syndtr
    cd syndtr
    git clone https://github.com/syndtr/goleveldb.git

Step 12. Clone golang/snappy.
  In Terminal enter:
    cd /usr/local/Cellar/go/1.10/libexec/src/vendor/github.com/golang/
    git clone https://github.com/golang/snappy.git

Step 13. Go to Elastos.ELA folder.
  In Terminal enter:
    cd /usr/local/Cellar/go/1.10/libexec/src/elastos.ela

Step 14. Make.
  In Terminal enter:
    make
    
 
