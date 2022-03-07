import unittest
import psycopg2
from psycopg2.policies import ClusterAwareLoadBalancer as cs
import os
from psycopg2._psycopg import OperationalError

class TestMisc(unittest.TestCase):
    def setupfortest1(self):
        self.yb_path = os.getenv('YB_PATH')
        os.system(self.yb_path+'/bin/yb-ctl destroy')
        os.system(self.yb_path+'/bin/yugabyted start --ysql_port 5544')

    def setupfortest2(self):
        self.yb_path = os.getenv('YB_PATH')
        os.system(self.yb_path+'/bin/yb-ctl destroy')
        os.system(self.yb_path+'/bin/yb-ctl create --rf 3 ')
        
    def cleanup(self):
        os.system(self.yb_path+'/bin/yb-ctl destroy')
        os.system(self.yb_path+'/bin/yugabyted destroy')

    def test_1(self):
        self.setupfortest1()
        def conn():
            conn = psycopg2.connect("user = yugabyte password=yugabyte host=localhost dbname=yugabyte load_balance=True")
        self.assertRaises(psycopg2.OperationalError,conn )
        self.cleanup()

    def test_2(self):
        self.setupfortest2()
        try:
            conn = psycopg2.connect("user = yugabyte password=yugabyte host=localhost dbname=yugabyte load_balance=True")
            conn.close()
        except OperationalError as e:
            self.fail("Test should have passed, but raised an OperationalError")
        finally:
            self.cleanup()

if __name__ == '__main__':
    unittest.main()