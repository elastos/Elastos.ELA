rm -rf *.8 *.o *.out *.6
find . -name "config.json" | sed 's#^#rm -fr #g' | sh
find . -name "Chain_UnitTest" | sed 's#^#rm -fr #g' | sh
find . -name "Chain_WhiteBox" | sed 's#^#rm -fr #g' | sh
find . -name "Logs" | sed 's#^#rm -fr #g' | sh
find . -name "ArbiterLogs" | sed 's#^#rm -fr #g' | sh
find . -name "DposEvent" | sed 's#^#rm -fr #g' | sh
