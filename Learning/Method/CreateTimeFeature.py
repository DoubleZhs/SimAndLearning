import pandas as pd
import numpy as np
import os
import shutil
from tqdm import tqdm
from joblib import Parallel, delayed


class CreateTimeFeature:
    def __init__(self,   
                 time_windows,
                 time_gap,
                 target,
                 time_feature_group,
                 ignore_first_day=True) -> None:
        """
        time_windows : 时序特征的个数，即需要计算前time_windows个平均旅行时间
        time_gap : 计算平均旅行时间的时间窗口大小
        time_feature_group : 一个列表，用于计算平均旅行时间的分组，可以是OD Route ， 也可以是OD Dig ，还可以加上别的特征。
        ignore_first_day : 是否忽视第一天
        """
        self.time_windows = time_windows
        self.time_gap = time_gap
        self.target = target
        self.time_feature_group = time_feature_group
        self.ignore_first_day = ignore_first_day


    # def __process_group(self, group, time_windows, time_gap, target):
    #     group = group.sort_values(by='In Time')
    #     group = group.set_index('In Time')
    #     min_actualInTime_total = group.index.min()
    #     new_group = group[group.index >= (min_actualInTime_total + time_windows * time_gap)].copy()
    #     # 为均值和标准差创建新列
    #     for tg in target:
    #         for i in range(1, time_windows + 1):
    #             lower_bound = new_group.index - time_gap * i
    #             upper_bound = new_group.index - time_gap * (i - 1)

    #             # 计算均值
    #             new_group[f'mean_{tg.replace(" ", "")}_before_{i}'] = [
    #                 round(group[(group.index < ub) & (group.index >= lb)][tg].mean(), 3)
    #                 for lb, ub in zip(lower_bound, upper_bound)
    #             ]
    #             # 计算标准差
    #             new_group[f'std_{tg.replace(" ", "")}_before_{i}'] = [
    #                 round(group[(group.index < ub) & (group.index >= lb)][tg].std(), 3)
    #                 for lb, ub in zip(lower_bound, upper_bound)
    #             ]
    #     new_group1 = new_group.ffill()
    #     new_group2 = new_group.bfill()
    #     for tg in target:
    #         for i in range(1, time_windows + 1):
    #             new_group[f'mean_{tg.replace(" ", "")}_before_{i}'] = (new_group1[f'mean_{tg.replace(" ", "")}_before_{i}'] 
    #                                                                 + new_group2[f'mean_{tg.replace(" ", "")}_before_{i}']) / 2
    #             new_group[f'std_{tg.replace(" ", "")}_before_{i}'] = (new_group1[f'std_{tg.replace(" ", "")}_before_{i}'] 
    #                                                                 + new_group2[f'std_{tg.replace(" ", "")}_before_{i}']) / 2
    #     return new_group.reset_index()
    def __process_group(self, group, time_windows, time_gap, target):
        group = group.sort_values(by='In Time')
        group = group.set_index('In Time')
        min_actualInTime_total = group.index.min()
        new_group = group[group.index >= (min_actualInTime_total + time_windows * time_gap)].copy()
        
        # 一天的时间步长
        day_steps = 57600  
        max_tries = 5  # 最多尝试n天
        
        def find_data_with_backtracking(lb, ub, tg):
            """
            尝试从当前窗口 [lb, ub) 获得数据，如果为空则尝试往前1天、2天、3天查找数据。
            """
            # 首先尝试当天数据
            intervals_to_try = [(lb - day_steps*k, ub - day_steps*k) for k in range(max_tries+1)]
            for (new_lb, new_ub) in intervals_to_try:
                subset = group[(group.index < new_ub) & (group.index >= new_lb)][tg]
                if not subset.empty:
                    return subset.mean(), subset.std()
            # 若仍无数据，则返回np.nan
            return np.nan, np.nan

        # 为新特征列准备容器（避免重复计算）
        # 结构为： { (tg, i): (mean_list, std_list) }
        result_storage = {}
        for tg in target:
            for i in range(1, time_windows + 1):
                result_storage[(tg, i)] = {"mean": [], "std": []}

        # 针对 new_group 的每一行，计算对应时间窗口的均值和标准差
        for idx in new_group.index:
            for tg in target:
                for i in range(1, time_windows + 1):
                    lower_bound = idx - time_gap * i
                    upper_bound = idx - time_gap * (i - 1)
                    mean_val, std_val = find_data_with_backtracking(lower_bound, upper_bound, tg)
                    result_storage[(tg, i)]["mean"].append(round(mean_val, 3) if not np.isnan(mean_val) else np.nan)
                    result_storage[(tg, i)]["std"].append(round(std_val, 3) if not np.isnan(std_val) else np.nan)

        # 将结果填入new_group中
        for tg in target:
            for i in range(1, time_windows + 1):
                new_group[f'mean_{tg.replace(" ","")}_before_{i}'] = result_storage[(tg, i)]["mean"]
                new_group[f'std_{tg.replace(" ","")}_before_{i}'] = result_storage[(tg, i)]["std"]

        # 使用前向填充和后向填充的中和策略
        new_group1 = new_group.ffill()
        new_group2 = new_group.bfill()
        for tg in target:
            for i in range(1, time_windows + 1):
                m_col = f'mean_{tg.replace(" ","")}_before_{i}'
                s_col = f'std_{tg.replace(" ","")}_before_{i}'
                new_group[m_col] = (new_group1[m_col] + new_group2[m_col]) / 2
                new_group[s_col] = (new_group1[s_col] + new_group2[s_col]) / 2

        return new_group.reset_index()

    def __featureEngineering(self, df: pd.DataFrame):

        def process_group_wrapper(group):
            return self.__process_group(group, self.time_windows, self.time_gap, self.target)
        
        groups = [group for name, group in df.groupby(self.time_feature_group)]
        processed_groups = Parallel(n_jobs=-1)(delayed(process_group_wrapper)(group) for group in tqdm(groups))
        new_df = pd.concat(processed_groups)
        if self.ignore_first_day:
            new_df = new_df[new_df['Date'] > 0].reset_index(drop=True)
        return new_df


    def create_time_feature(self, sub_path, num_DiDi_lst, pro_Autonavi_lst):
        """
        常规的特征生成，只抽pro不抽n
        sub_path : 文件夹名称
        num_DiDi_lst : 需要使用的DiDi数列表
        pro_Autonavi_lst : 每个DiDi数下，需要生成的pro列表
        """

        if not os.path.exists(f"Learning/TimeFeature/{sub_path}"):
            os.makedirs(f"Learning/TimeFeature/{sub_path}")
            
        print("生成时序特征")
        for num_DiDi in num_DiDi_lst:

            df = pd.read_csv(f"Learning/Feature/{sub_path}/VehicleData_n{num_DiDi}.csv")
            # 先对全量的做一遍
            total_df = self.__featureEngineering(df)
            total_df.to_csv(f"Learning/TimeFeature/{sub_path}/VehicleData_n{num_DiDi}_total.csv", index=None)
            # 对滴滴的做一遍
            DiDi_df = df[df['ClosedVehicle'] == True]
            DiDi_df = self.__featureEngineering(DiDi_df) 
            DiDi_df.to_csv(f"Learning/TimeFeature/{sub_path}/VehicleData_n{num_DiDi}.csv", index=None)
            
            # 最后对高德的做一遍
            for j in tqdm(range(len(pro_Autonavi_lst))):

                pro_Autonavi = round(pro_Autonavi_lst[j], 3)               
                Autonavi_df = df[(df['ClosedVehicle'] == False) & (df['Tag'] <= pro_Autonavi)]
                Autonavi_df = self.__featureEngineering(Autonavi_df)
                Autonavi_df.to_csv(f"Learning/TimeFeature/{sub_path}/VehicleData_n{num_DiDi}_pro{pro_Autonavi}.csv", index=None)

    def create_time_feature_sample(self, sub_path, n, num_DiDi_lst, pro_Autonavi_lst):
        """
        只拿滴滴数为n的数据集，即抽num_DiDi，又抽Pro
        sub_path : 文件夹名称
        n : 所使用仿真的滴滴数
        num_DiDi_lst : 需要生成的DiDi数列表
        pro_Autonavi_lst : 需要生成的pro列表
        """

        if not os.path.exists(f"Learning/TimeFeatureBySampling/{sub_path}"):
            os.makedirs(f"Learning/TimeFeatureBySampling/{sub_path}")
        
        # 读取唯一的文件
        df = pd.read_csv(f"Learning/Feature/{sub_path}/VehicleData_n{n}.csv")
        
        print("生成时序特征")

        # 检查是否已经在常规处理中存在
        if os.path.exists(f"Learning/TimeFeature/{sub_path}/VehicleData_n{n}_total.csv"):
            # 复制文件并重命名
            shutil.copy(f"Learning/TimeFeature/{sub_path}/VehicleData_n{n}_total.csv", f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{n}_total.csv")
        else:
            # 如果不存在，先对全量的做一遍
            total_df = self.__featureEngineering(df)
            total_df.to_csv(f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{n}_total.csv", index=None)

        # 抽取生成滴滴数据
        for num_DiDi in tqdm(num_DiDi_lst):
            if os.path.exists(f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{n}_num_DiDi{num_DiDi}.csv"):
                continue
            # 拿出对应部分的滴滴数据
            DiDi_df = df[df['Vehicle ID'] <= num_DiDi]
            DiDi_df = self.__featureEngineering(DiDi_df) 
            DiDi_df.to_csv(f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{n}_num_DiDi{num_DiDi}.csv", index=None)

        for pro_Autonavi in tqdm(pro_Autonavi_lst):
            pro_Autonavi = round(pro_Autonavi, 3)
            if os.path.exists(f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{n}_pro{pro_Autonavi}.csv"):
                continue
            if os.path.exists(f"Learning/TimeFeature/{sub_path}/VehicleData_n{n}_pro{pro_Autonavi}.csv"):
                # 复制文件并重命名
                shutil.copy(f"Learning/TimeFeature/{sub_path}/VehicleData_n{n}_pro{pro_Autonavi}.csv", f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{n}_pro{pro_Autonavi}.csv")
            else:
                Autonavi_df = df[(df['ClosedVehicle'] == False) & (df['Tag'] <= pro_Autonavi)]
                Autonavi_df = self.__featureEngineering(Autonavi_df)
                Autonavi_df.to_csv(f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{n}_pro{pro_Autonavi}.csv", index=None)