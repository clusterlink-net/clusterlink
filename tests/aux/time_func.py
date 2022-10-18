################################################################
#Name: time_func 
#Desc: Functions to calculate tests duration
################################################################
import time
from datetime import datetime

def test_start_time():
    start_time = datetime.now()
    start_time_s = start_time.strftime("%H:%M:%S")
    return start_time
    
def test_end_time(start_time):
    end_time    = datetime.now()
    end_time_s  = end_time.strftime("%H:%M:%S")
    test_time_s =end_time-start_time
    print("Test start {} Test end {} total test time {}".format(start_time,end_time, test_time_s))