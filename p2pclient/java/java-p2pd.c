#include <jni.h>
#include "java-p2pd.h"
#include "go-p2pd.h"

JNIEXPORT void JNICALL Java_p2pd_startDaemon 
(JNIEnv *env, jclass cl){
    startDaemon();
}

JNIEXPORT void JNICALL Java_p2pd_stopDaemon
(JNIEnv *env, jclass cl){
    stopDaemon();
}