import argparse
import os
import datetime

def process_line(colon_parts, row_offset, d):
    """
    Extract the details we want from each relevant line in the log
    """
    key = colon_parts[4].strip()
    hour = int(colon_parts[0].split(" ")[2])
    min = int(colon_parts[1])
    sec = int(float(colon_parts[2].split(" ")[0]))
    microsecs = int((float(colon_parts[2].split(" ")[0]) - float(sec)) * 1000000)

    t = datetime.datetime(2020,1,1,hour,min,sec,microsecs)
    if not d.get(key):
        d[key] = {}
    d[key][row_offset] = t

def get_sorted_microsecond_intervals(d, start_key, end_key, key_offset=1):
    """
    Returns a sorted list (low -> high) of the processing cost in microseconds,
    along with the offset in the log file where this cost is found (for easy lookup)
    """
    start = d[start_key]
    end = d[end_key]

    intervals = []

    for k, start_time in start.items():
        end_time = end.get(k+key_offset, None)
        if end_time:
            intervals.append((k, 1000000*(end_time-start_time).total_seconds()))
        else:
            print(k, k+key_offset)

    intervals.sort(key=lambda tup: tup[1])

    return intervals

def get_interval_stats(d, start_key, end_key, key_offset=1):
    """
    Generate the stats from the timing data.  
    Note that start/end keys may not be adjacent in the log file, so allow specification of the offset
    """
    def get_percentile(l, p):
        offset = int(len(l)*float(p)/100)
        if offset > len(l)-1:
            offset = len(l)-1
        elif offset < 0:
            offset = 0
        return l[offset][1]

    def get_mean(l):
        sum = 0.0
        for v in l:
            sum = sum + v[1]
        return sum / len(l)

    intervals = get_sorted_microsecond_intervals(d, start_key, end_key, key_offset=key_offset)

    return [len(intervals), get_mean(intervals)] + \
        [get_percentile(intervals, p) for p in [0, 50, 60, 70, 75, 80, 85, 90, 95, 99, 100]] 

def pretty_print(title, stats):
    """
    Easy to read output
    """
    
    print(f'Measuring {title} ({stats[0]} items found in log):')

    offset = 0
    for t in ["Mean", "Min", "p50", "p60", "p70", "p75", "p80", "p85", "p90", "p95", "p99", "max"]:
        offset = offset + 1
        print(f'\t{t}:\t{str(int(stats[offset])).rjust(8, " ")}')

def get_log_contents(file_name, show_read, show_decryption, show_compression, show_overall):
    """
    Read the log, and print out useful stats
    """
    d = {}

    # Only process lines that log work starting/ending within a request
    with open(file_name) as f:
        row_offset = 0
        for s in f.readlines():
            row_offset = row_offset + 1
            parts = s.split(":")
            if len(parts) == 5:
                process_line(parts, row_offset, d)

    if show_read:
        pretty_print("Reading", get_interval_stats(d, "Reading from Disk", "Reading from Disk completed"))

    if show_decryption:
        pretty_print("Decryption", get_interval_stats(d, "Decrypting", "Decrypted"))
    
    if show_compression:
        pretty_print("Decompresssion", get_interval_stats(d, "Uncompressing", "Uncompressed"))
    
    if show_overall:
        pretty_print("Overall Retrieval Time", get_interval_stats(d, "Retrieving", "Completed retrieval", 7))

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Analyse DataProxy logfile timings')
    parser.add_argument('--c', dest='c', action='store_true', help='Compression stats')
    parser.add_argument('--d', dest='d', action='store_true', help='Decryption stats')
    parser.add_argument('--r', dest='r', action='store_true', help='Read stats')
    parser.add_argument('--o', dest='o', action='store_true', help='Overall retrieval stats')
    parser.add_argument('logfile', type=str, help='Location of the logfile to be parsed')

    args = parser.parse_args()

    get_log_contents(args.logfile, args.r, args.d, args.c, args.o)
