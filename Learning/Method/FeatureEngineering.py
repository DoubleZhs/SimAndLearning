import pandas as pd
import numpy as np
import os



class FeatureEngineering:
    def __init__(self,num_of_cells,gaps_between_traffic_lights,gaps_between_digs,min_dist,group_dist):
        """
        num_of_cells: 元胞数量
        gaps_between_traffic_lights: 交通灯之间的元胞数
        gaps_between_digs: 自定义的OD划分元胞数
        min_dist: 最小行程距离（单位为元胞数），小于该距离的行程会被丢弃
        group_dist: 自定义的行程分组距离（单位为元元胞数），用于生成 Distance Dig
        """
        self.num_of_cells = num_of_cells
        self.gaps_between_traffic_lights = gaps_between_traffic_lights
        self.gaps_between_digs = gaps_between_digs
        self.min_dist = min_dist
        self.group_dist = group_dist
        self.num_of_traffic_lights = self.num_of_cells // self.gaps_between_traffic_lights
        self.num_of_digs = self.num_of_cells // self.gaps_between_digs
 
    def run(self,sub_path,timestamp,n):
        """
        sub_path: 数据存储文件夹路径
        timestamp: 时间戳
        n: 滴滴数
        """
        df = pd.read_csv(f"Learning/Data/{sub_path}/{timestamp}_{n}_VehicleData.csv",encoding='utf-8')
        df['Travel Time'] = df['Arrival Time'] - df['In Time'] 
        df = df[df['Travel Time']>0].reset_index(drop=True)
        
        travel_time_mean = df['Travel Time'].mean()
        travel_time_std = df['Travel Time'].std()
        # 标准化
        df['Travel Time Standardized'] = (df['Travel Time'] - travel_time_mean) / travel_time_std
        # 对数变换
        df['Travel Time Log'] = np.log1p(df['Travel Time'])

        # 增加日期
        print("时间特征")
        df['Date'] = df['In Time'] // 57600
        df['Actual In Time'] = df['In Time'] % 57600  
        df['Actual Arrival Time'] = df['Arrival Time'] % 57600
        df['Hour'] = df['Actual In Time'] // 2400
        df['Quarter'] = df['Actual In Time'] // 400


        # 是否处于早高峰/晚高峰
        # print("增加早高峰/晚高峰")
        # df['Early Commute'] = ((df['Hour'] >= 7) & (df['Hour'] <= 10)).astype(int)
        # df['Late Commute'] = ((df['Hour'] >= 17) & (df['Hour'] <= 20)).astype(int) 
        
        print('OD 分区')
        gaps_list = [self.gaps_between_digs, self.gaps_between_digs * 5, self.gaps_between_digs * 10]

        for i, gap in enumerate(gaps_list):
            num_of_digs = self.num_of_cells // gap
            df[f'O_Dig_{i}'] = np.where(
                df['Origin'] == 0,
                num_of_digs - 1,
                df['Origin'] // gap
            )
            df[f'D_Dig_{i}'] = np.where(
                df['Destination'] == 0,
                num_of_digs - 1,
                df['Destination'] // gap
            )
            df[f'OD_Dig_{i}'] = df.apply(lambda row: f"{row[f'O_Dig_{i}']}_{row[f'D_Dig_{i}']}", axis=1)

        # 在min_dist内的OD pair不被统计
        print("删除极短距离行程")
        print(df[df['PathLength'] <= self.min_dist].shape[0])
        df = df[df['PathLength'] > self.min_dist].reset_index(drop=True)

        # 为距离增加分组信息
        print('Distance 特征')
        distance_gaps_list = [self.group_dist, self.group_dist * 4, self.group_dist * 8]

        for i, gap in enumerate(distance_gaps_list):
            df[f'Distance_Dig_{i}'] = df['PathLength'] // gap

        # 增加红绿灯信息
        print("增加红绿灯信息")
        df['Traffic Light Count'] = 0
        for i in range(self.num_of_traffic_lights):
            df[f'Traffic Light {i}'] = 0

        # 红绿灯位置
        traffic_lights = [i * self.gaps_between_traffic_lights for i in range(self.num_of_traffic_lights)]

        def calculate_traffic_lights(row):
            origin = row['Origin']
            destination = row['Destination']
            lights_passed = []
            count = 0

            if origin < destination:
                for light in traffic_lights:
                    if origin < light <= destination:
                        lights_passed.append(light)
                        count += 1
            else:
                for light in traffic_lights:
                    if origin < light or light <= destination:
                        lights_passed.append(light)
                        count += 1

            return count, lights_passed

        df[['Traffic Light Count', 'Traffic Lights Passed']] = df.apply(lambda row: calculate_traffic_lights(row), axis=1, result_type='expand')

        for i in range(self.num_of_traffic_lights):
            df[f'Traffic Light {i}'] = df['Traffic Lights Passed'].apply(lambda x: 1 if i * self.gaps_between_traffic_lights in x else 0)
        # 删除临时列
        df.drop(columns=['Traffic Lights Passed'], inplace=True)

        if not os.path.exists(f'Learning/Feature/{sub_path}'):
            os.makedirs(f'Learning/Feature/{sub_path}')
        df.to_csv(f"Learning/Feature/{sub_path}/VehicleData_n{n}.csv",encoding='utf-8',index=None)


