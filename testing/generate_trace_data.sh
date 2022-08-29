START_TIME=$(date +%s)
HEX_TIME=$(printf '%x\n' $START_TIME)
GUID=$(dd if=/dev/random bs=12 count=1 2>/dev/null | od -An -tx1 | tr -d ' \t\n')
TRACE_ID="1-$HEX_TIME-$GUID"
SEGMENT_ID=$(dd if=/dev/random bs=8 count=1 2>/dev/null | od -An -tx1 | tr -d ' \t\n')
SEGMENT_DOC="{\"trace_id\": \"$TRACE_ID\", \"id\": \"$SEGMENT_ID\", \"start_time\": $START_TIME, \"in_progress\": true, \"name\": \"XRay-Daemon-Test\"}"
HEADER='{"format": "json", "version": 1}'
TRACE_DATA="$HEADER\n$SEGMENT_DOC"
echo "$HEADER" > trace_document.txt
echo "$SEGMENT_DOC" >> trace_document.txt

echo "TRACE_ID=$TRACE_ID" >> $GITHUB_ENV
