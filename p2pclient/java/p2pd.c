#include <jni.h>
#include "p2pd.h"
#include "libp2pd.h"

JNIEXPORT void JNICALL Java_p2pd_startD 
(JNIEnv *env, jclass cl){
    startD();
}

JNIEXPORT void JNICALL Java_p2pd_stopD
(JNIEnv *env, jclass cl){
    stopD();
}