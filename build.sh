TYPE="deb"
PACKAGE_NAME="puckgo"
VERSION="1.0.0"
VENDOR="puck"
#BUILD_PATH=/tmp/puckgo
BUILD_PATH=package
SCRIPTS_PATH=./scripts
BUILD_BIN_PATH=$BUILD_PATH/usr/sbin
BUILD_ETC_PATH=$BUILD_PATH/usr/etc/puckgo/
BUILD_LOG_PATH=$BUILD_PATH/var/log/puckgo/
BUILD_SYSTEMD_PATH=$BUILD_PATH/usr/lib/systemd/system/

#rm -fr $BUILD_PATH
#mkdir $BUILD_PATH
#mkdir -p $BUILD_BIN_PATH
#mkdir -p $BUILD_ETC_PATH
#mkdir -p $BUILD_LOG_PATH
#mkdir -p $BUILD_SYSTEMD_PATH

go build -ldflags "-w -s"
cp puckgo $BUILD_BIN_PATH

#chmod 700 ${BUILD_PATH}/*
fpm -f -s dir \
    -t $TYPE \
    -n $PACKAGE_NAME \
    -v $VERSION \
    -C $BUILD_PATH \
    -p . \
    --prefix / \
    --vendor $VENDOR \
    --deb-no-default-config-files \
    #--post-install $SCRIPTS_PATH/post-install \
    #--pre-uninstall $SCRIPTS_PATH/pre-uninstall \
    #--post-uninstall $SCRIPTS_PATH/post-uninstall
