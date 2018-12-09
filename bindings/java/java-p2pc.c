#include "java-p2pc.h"
#include "../go-p2pc.h"

JNIEXPORT void JNICALL Java_p2pc_startClient (JNIEnv *jenv, jclass jcls, jstring jarg1){
  char *arg1 = (char *) 0 ;
  (void)jenv;
  (void)jcls;
  arg1 = 0;
  if (jarg1) {
    arg1 = (char *)(*jenv)->GetStringUTFChars(jenv, jarg1, 0);
    if (!arg1) return ;
  }
  startClient(arg1);
  if (arg1) (*jenv)->ReleaseStringUTFChars(jenv, jarg1, (const char *)arg1);
}