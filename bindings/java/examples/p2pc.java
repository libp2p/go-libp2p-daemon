public class p2pc {
    private static final String NAME = "p2pc"; 
    public static native void startClient(String arg1);
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
        startClient(parsedArgs);
    }
}