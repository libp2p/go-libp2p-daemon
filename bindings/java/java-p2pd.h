#include <jni.h>

#ifndef _Included_p2pd
#define _Included_p2pd
#ifdef __cplusplus
extern "C" {
#endif

JNIEXPORT void JNICALL Java_p2pd_startDaemon (JNIEnv *, jclass, jstring);

JNIEXPORT void JNICALL Java_p2pd_stopDaemon (JNIEnv *, jclass);

#ifdef __cplusplus
}
#endif
#endif
