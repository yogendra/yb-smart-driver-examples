import unittest
import psycopg2
from psycopg2.policies import ClusterAwareLoadBalancer as cs
from psycopg2 import pool
import queue
from threading import Thread
import os
import time

que = queue.Queue()

class TestUniformLoadBalancer(unittest.TestCase):

    yb_path = ''

    def setup(self):
        self.yb_path = os.getenv('YB_PATH')
        os.system(self.yb_path+'/bin/yb-ctl destroy')
        os.system(self.yb_path+'/bin/yb-ctl create --rf 3 ')

    def create_conns(self, numConns):
        conns = []
        for i in range(0, numConns):
            conn = psycopg2.connect(user = 'yugabyte', password='yugabyte', host = '127.0.0.1', port = '5433', database = 'yugabyte', load_balance='True')
            conns.append(conn)
        return conns

    def cleanup(self, conns):
        for conn in conns:
            conn.close()
        conns.clear()
        os.system(self.yb_path+'/bin/yb-ctl destroy')
    
    def test_2(self):
        self.setup()
        print("Creating 12 connections")
        conns = self.create_conns(12)
        obj = cs()
        numConnMap = obj.hostToNumConnMap
        for host,numconn in numConnMap.items():
            self.assertEqual(numconn, 4)
        print('Cleaning up...')
        self.cleanup(conns)
    
    def test_3(self):
        self.setup()
        conns = []
        t1 = Thread(target=lambda q, arg1: q.put(self.create_conns(arg1)), args=(que, 6))
        t2 = Thread(target=lambda q, arg1: q.put(self.create_conns(arg1)), args=(que, 6))

        t1.start()
        t2.start()

        t1.join()
        t2.join()

        while not que.empty():
            result = que.get()
            conns = conns + result
        obj = cs()
        numConnMap = obj.hostToNumConnMap
        for host,numconn in numConnMap.items():
            self.assertEqual(numconn, 4)
        print('Cleaning up...')
        self.cleanup(conns)

    def test_4(self):
        self.setup()
        os.system(self.yb_path+'/bin/yb-ctl stop_node 2')
        time.sleep(5)
        print("Creating 10 connections")
        conns = self.create_conns(10)
        obj = cs()
        numConnMap = obj.hostToNumConnMap
        for host,numconn in numConnMap.items():
            if host == '127.0.0.2' :
                self.assertEqual(numconn, 0)
            else : 
                self.assertEqual(numconn, 5)
        self.cleanup(conns)
    
    def test_5(self):
        self.setup()
        print("Creating 12 connections")
        conns = self.create_conns(12)
        os.system(self.yb_path+'/bin/yb-ctl add_node')
        time.sleep(300)
        print("Creating 8 connections")
        conns = conns + self.create_conns(8)
        obj = cs()
        numConnMap = obj.hostToNumConnMap
        print(numConnMap)
        for host,numconn in numConnMap.items():
            self.assertEqual(numconn, 5)
        self.cleanup(conns)

    def test_6(self):
        self.setup()
        print("Creating 12 connections")
        conns = self.create_conns(12)
        obj = cs()
        numConnMap = obj.hostToNumConnMap
        for host,numconn in numConnMap.items():
            self.assertEqual(numconn, 4)
        print('Creating 3 more connections with host=localhost')
        for i in range(3):
            conn = psycopg2.connect(user = 'yugabyte', password='yugabyte', host = 'localhost', port = '5433', database = 'yugabyte', load_balance='True')
            conns.append(conn)
        numConnMap = obj.hostToNumConnMap
        for host,numconn in numConnMap.items():
            self.assertEqual(numconn, 5)
        print('Cleaning up...')
        self.cleanup(conns)

    def test_7(self):
        self.setup()
        conns = []
        postgreSQL_pool = psycopg2.pool.SimpleConnectionPool(1, 20, user="yugabyte",
                                                         password="yugabyte",
                                                         host="127.0.0.1",
                                                         port="5433",
                                                         database="yugabyte",
                                                         load_balance="True")
        if (postgreSQL_pool):
            print("Connection pool created successfully")

        print('Creating 12 connections...')
        for i in range(12):
            conn = postgreSQL_pool.getconn()
            conns.append(conn)
        obj = cs()
        numConnMap = obj.hostToNumConnMap
        for host,numconn in numConnMap.items():
            self.assertEqual(numconn, 4)
        print('Cleaning up...')
        self.cleanup(conns)

if __name__ == '__main__':
    unittest.main()