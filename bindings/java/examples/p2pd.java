public class p2pd {
    private static final String NAME = "p2pd"; 
    public static native void startDaemon(String arg1);
    public static native void stopDaemon();
    static {
        try {
            
            System.loadLibrary ( NAME ) ;
            
        } catch (UnsatisfiedLinkError e) {
          System.err.println("Native code library failed to load.\n" + e);
          System.exit(1);
        }
    }
    public static void main(String[] args) {
        String parsedArgs = NAME;
        if( args.length > 0 ){
            parsedArgs += "|" + String.join("|", args);
        }
        startDaemon(parsedArgs);
    }
}