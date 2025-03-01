import os
import csv
import pickle
import math
import time
import cudf
import cupy
import numpy as np
import matplotlib.pyplot as plt

from xgboost import plot_importance
from sklearn.model_selection import train_test_split
from sklearn.metrics import mean_squared_error, mean_absolute_error, r2_score, mean_absolute_percentage_error

# Dask & Dask-CUDA
from dask.distributed import Client, as_completed
from dask_cuda import LocalCUDACluster
import dask_cudf
from xgboost.dask import DaskXGBRegressor

class LearningForRingBySampling:
    def __init__(self, sample_n, n_lst, pro_lst, date_lst, features, target):
        """
        初始化
        """
        self.sample_n = sample_n
        self.n_lst = n_lst
        self.pro_lst = pro_lst
        self.date_lst = date_lst
        self.features = features
        self.target = target

        # 启动Dask多GPU集群
        self.cluster = LocalCUDACluster()
        self.client = Client(self.cluster)
        print("Dask cluster started:", self.client)

    def search_params(self, X_train, y_train, X_val, y_val, n_iter=30):
        """
        使用Dask并行在GPU上进行参数搜索。
        """
        # 定义参数空间
        param_dist = {
            'n_estimators': list(np.arange(100, 2001, 100)),
            'max_depth': [3, 4, 5, 6, 7, 8, 9, 10],
            'min_child_weight': [1, 2, 3, 4, 5, 6],
            'gamma': [0.1, 0.2, 0.3, 0.4, 0.5, 0.6],
            'subsample': [0.6, 0.7, 0.8, 0.9],
            'colsample_bytree': [0.6, 0.7, 0.8, 0.9],
            'reg_alpha': [0.05, 0.1, 1, 2, 3],
            'reg_lambda': [0.05, 0.1, 1, 2, 3],
            'learning_rate': [0.01, 0.02, 0.05, 0.1, 0.15, 0.2],
        }

        keys = list(param_dist.keys())

        # 随机生成参数组合
        np.random.seed(42)
        param_combinations = []
        for _ in range(n_iter):
            params = {k: np.random.choice(v) for k, v in param_dist.items()}
            param_combinations.append(params)

        # 结果收集
        results = []

        # 定义训练函数
        def train_evaluate(params, X_train, y_train, X_val, y_val):
            # 动态设置 early_stopping_rounds 为 n_estimators 的 10%
            esr = max(1, int(params['n_estimators'] * 0.1))
            params_updated = params.copy()
            params_updated.update({
                'objective': 'reg:squarederror',
                'tree_method': 'gpu_hist',
                'enable_categorical': True,
                'eval_metric': 'mape',
                'predictor': 'gpu_predictor',
                'early_stopping_rounds': esr
            })

            # 初始化模型
            model = DaskXGBRegressor(**params_updated)
            model.client = self.client

            # 训练模型
            model.fit(X_train, y_train, eval_set=[(X_val, y_val)], verbose=False)

            # 预测
            y_pred = model.predict(X_val).compute()

            # 计算性能
            y_true = y_val.compute().to_pandas()
            y_pred = y_pred.to_pandas()
            performance = self.getPerformance(y_true, y_pred)

            return {'params': params_updated, 'performance': performance}

        # 提交所有参数组合的训练任务
        futures = []
        for params in param_combinations:
            futures.append(self.client.submit(train_evaluate, params, X_train, y_train, X_val, y_val))

        # 等待所有任务完成
        for future in as_completed(futures):
            result = future.result()
            results.append(result)
            print(f"Completed Params: {result['params']}, Performance: {result['performance']}")

        # 选择最佳参数（基于MAPE最低）
        best_result = min(results, key=lambda x: x['performance']['MAPE'])
        best_params = best_result['params']
        print("Best Parameters:", best_params)
        return best_params

    def getPerformance(self, y_true, y_pred):
        """
        计算回归性能指标
        """
        MSE = mean_squared_error(y_true, y_pred)
        RMSE = math.sqrt(MSE)
        mae = mean_absolute_error(y_true, y_pred)
        r2 = r2_score(y_true, y_pred)
        mape = mean_absolute_percentage_error(y_true, y_pred)
        return {'MSE': MSE, 'RMSE': RMSE, 'MAE': mae, 'R2': r2, 'MAPE': mape}

    def run(self, train_df, valid_df, test_df, total_test_df, is_search):
        """
        训练模型并进行预测
        """
        # 将数据转换为dask_cudf DataFrame
        dtrain = dask_cudf.from_cudf(train_df, npartitions=4)
        dvalid = dask_cudf.from_cudf(valid_df, npartitions=2)
        dtest = dask_cudf.from_cudf(test_df, npartitions=1)
        dtotal_test = dask_cudf.from_cudf(total_test_df, npartitions=1)

        X_train = dtrain[self.features]
        y_train = dtrain[self.target]
        X_val = dvalid[self.features]
        y_val = dvalid[self.target]

        # 参数搜索使用GPU并行
        if is_search:
            best_params = self.search_params(X_train, y_train, X_val, y_val, n_iter=30)
        else:
            # 无参数搜索时的默认参数
            esr = max(1, int(1400 * 0.1))  # 140
            best_params = {
                'objective': 'reg:squarederror',
                'tree_method': 'gpu_hist',
                'enable_categorical': True,
                'eval_metric': 'mape',
                'n_estimators': 1400,
                'learning_rate': 0.01,
                'max_depth': 10,
                'min_child_weight': 2,
                'gamma': 0.6,
                'subsample': 0.6,
                'colsample_bytree': 0.7,
                'reg_alpha': 3,
                'reg_lambda': 1,
                'predictor': 'gpu_predictor',
                'early_stopping_rounds': esr
            }

        # 初始化模型
        xgb_model = DaskXGBRegressor(**best_params)
        xgb_model.client = self.client

        # 训练模型
        xgb_model.fit(X_train, y_train, eval_set=[(X_val, y_val)], verbose=False)

        # 预测
        test_pred = xgb_model.predict(dtest[self.features]).compute()
        total_test_pred = xgb_model.predict(dtotal_test[self.features]).compute()

        test_df['Predicted Travel Time'] = test_pred
        total_test_df['Predicted Travel Time'] = total_test_pred

        # 计算性能
        performance = self.getPerformance(test_df[self.target].to_pandas(), test_df['Predicted Travel Time'].to_pandas())
        total_performance = self.getPerformance(total_test_df[self.target].to_pandas(), total_test_df['Predicted Travel Time'].to_pandas())

        return xgb_model, test_df, total_test_df, performance, total_performance

    def learning(self, sub_path, traindaynote, testnote, is_search=False, k=None, days_set=None):
        """
        主学习流程
        """
        total_starttime = time.time()

        # 将note组合
        note = f"{sub_path}&{traindaynote}&{testnote}&sample_n{self.sample_n}&is_search_{is_search}"

        # 创建路径
        os.makedirs(f'Learning/Model/{note}', exist_ok=True)
        os.makedirs(f'Learning/PredictResult/{note}', exist_ok=True)
        os.makedirs(f'Learning/Logs', exist_ok=True)
        os.makedirs(f'Learning/Performance', exist_ok=True)

        # 新建日志
        log_path = f"Learning/Logs/logs_{note}.txt"
        with open(log_path, 'w') as file:
            pass

        # 新建performance
        performance_path = f"Learning/Performance/performance_{note}.csv"
        fieldnames = ['MSE','RMSE','MAE','R2','MAPE','Type','Note','Pro','n','date']
        with open(performance_path, 'w') as file:
            writer = csv.DictWriter(file, fieldnames=fieldnames)
            writer.writeheader()

        # 使用dask_cudf读取全集数据，移除了 encoding 参数
        data_df = dask_cudf.read_csv(
            f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{self.sample_n}_total.csv"
        ).compute()

        # 处理DiDi数据
        for n in self.n_lst:
            DiDi_df = dask_cudf.read_csv(
                f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{self.sample_n}_num_DiDi{n}.csv"
            ).compute()
            for date in self.date_lst:
                print(f"-----------------------------------n : {n} ; Date: {date} ; DiDi Result:-----------------------------------")
                with open(log_path, 'a') as file:
                    file.write(f"-----------------------------------n : {n} ; Date: {date} ; DiDi Result:-----------------------------------\n")

                start_time = time.time()

                # 根据测试策略选择训练/测试集
                if testnote == '测试周期最后':
                    if date == self.date_lst[-1]:
                        continue
                    # 获取测试日期
                    test_date = None
                    for days in days_set:
                        if date < days[1]:
                            test_date = days[1]
                            break
                    if not test_date:
                        test_date = self.date_lst[-1]

                    # 选择训练和验证数据
                    if traindaynote == '天数无上限':
                        DiDi_train_valid_df = DiDi_df[DiDi_df['Date'] <= date]
                    elif traindaynote == f'天数上限{k}':
                        DiDi_train_valid_df = DiDi_df[(DiDi_df['Date'] >= date - k + 1) & (DiDi_df['Date'] <= date)]

                    DiDi_train_df, DiDi_valid_df = train_test_split(DiDi_train_valid_df, test_size=0.2, random_state=42, shuffle=True)
                    DiDi_test_df = DiDi_df[DiDi_df['Date'] == test_date].reset_index(drop=True)
                    DiDi_total_test_df = data_df[data_df['Date'] == test_date].reset_index(drop=True)

                elif testnote == '测试周期抽取':
                    # 获取测试日期范围
                    test_date1, test_date2 = None, None
                    for days in days_set:
                        if date < days[1] + 1:
                            test_date1, test_date2 = days[0], days[1]
                            break
                    if not test_date1 or not test_date2:
                        test_date1, test_date2 = days_set[-1]

                    # 拆分数据
                    DiDi_df1, DiDi_df2 = train_test_split(DiDi_df, test_size=0.2, random_state=42, shuffle=True)
                    if traindaynote == '天数无上限':
                        DiDi_train_valid_df = DiDi_df1[DiDi_df1['Date'] <= date]
                    elif traindaynote == f'天数上限{k}':
                        DiDi_train_valid_df = DiDi_df1[(DiDi_df1['Date'] >= date - k + 1) & (DiDi_df1['Date'] <= date)]

                    DiDi_train_df, DiDi_valid_df = train_test_split(DiDi_train_valid_df, test_size=0.2, random_state=42, shuffle=True)
                    DiDi_test_df = DiDi_df2[(DiDi_df2['Date'] >= test_date1) & (DiDi_df2['Date'] <= test_date2)].reset_index(drop=True)
                    DiDi_total_test_df = data_df[(data_df['Date'] >= test_date1) & (data_df['Date'] <= test_date2)].reset_index(drop=True)

                elif testnote == '测试为下一天':
                    if date == self.date_lst[-1]:
                        continue
                    # 选择训练和验证数据
                    if traindaynote == '天数无上限':
                        DiDi_train_valid_df = DiDi_df[DiDi_df['Date'] <= date]
                    elif traindaynote == f'天数上限{k}':
                        DiDi_train_valid_df = DiDi_df[(DiDi_df['Date'] >= date - k + 1) & (DiDi_df['Date'] <= date)]

                    DiDi_train_df, DiDi_valid_df = train_test_split(DiDi_train_valid_df, test_size=0.2, random_state=42, shuffle=True)
                    DiDi_test_df = DiDi_df[DiDi_df['Date'] == date + 1].reset_index(drop=True)
                    DiDi_total_test_df = data_df[data_df['Date'] == date + 1].reset_index(drop=True)

                # 训练模型并进行预测
                DiDi_model, DiDi_test_df, DiDi_total_test_df, DiDi_performance, DiDi_total_performance = self.run(
                    DiDi_train_df, DiDi_valid_df, DiDi_test_df, DiDi_total_test_df, is_search
                )

                # 保存模型
                with open(f'Learning/Model/{note}/n{self.sample_n}_num_DiDi{n}_date{date}_xgb_model.pkl', 'wb') as file:
                    pickle.dump(DiDi_model, file)

                # 添加性能信息
                DiDi_performance['Type'] = 'DiDi'
                DiDi_performance['Note'] = 'pure'
                DiDi_performance['Pro'] = DiDi_df.shape[0] / data_df.shape[0]
                DiDi_performance['n'] = n
                DiDi_performance['date'] = date

                DiDi_total_performance['Type'] = 'DiDi'
                DiDi_total_performance['Note'] = 'total'
                DiDi_total_performance['Pro'] = DiDi_df.shape[0] / data_df.shape[0]
                DiDi_total_performance['n'] = n
                DiDi_total_performance['date'] = date

                end_time = time.time()

                print(f"pure : MAPE {DiDi_performance['MAPE']}")
                print(f"total: MAPE {DiDi_total_performance['MAPE']}")
                print(f"Time : {end_time - start_time} seconds")

                with open(log_path, 'a') as file:
                    file.write(f"pure : MAPE {DiDi_performance['MAPE']}\n")
                    file.write(f"total: MAPE {DiDi_total_performance['MAPE']}\n")
                    file.write(f"Time : {end_time - start_time} seconds\n")

                with open(performance_path, 'a') as file:
                    writer = csv.DictWriter(file, fieldnames=fieldnames)
                    writer.writerow(DiDi_performance)
                    writer.writerow(DiDi_total_performance)

                # 保存预测结果
                DiDi_test_df.to_cudf().to_csv(f"Learning/PredictResult/{note}/DiDi_predictions_n{self.sample_n}_num_DiDi{n}_date{date}.csv", index=False)

        # 处理Autonavi数据
        for Autonavi_pro in self.pro_lst:
            Autonavi_pro = round(Autonavi_pro, 3)
            Autonavi_df = dask_cudf.read_csv(
                f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{self.sample_n}_pro{Autonavi_pro}.csv"
            ).compute()
            for date in self.date_lst:
                print(f"-----------------------------------Date: {date} ; Autonavi {Autonavi_pro} Result:-----------------------------------")
                with open(log_path, 'a') as file:
                    file.write(f"-----------------------------------Date: {date} ; Autonavi {Autonavi_pro} Result:-----------------------------------\n")

                start_time = time.time()

                # 根据测试策略选择训练/测试集
                if testnote == '测试周期最后':
                    if date == self.date_lst[-1]:
                        continue
                    # 获取测试日期
                    test_date = None
                    for days in days_set:
                        if date < days[1]:
                            test_date = days[1]
                            break
                    if not test_date:
                        test_date = self.date_lst[-1]

                    # 选择训练和验证数据
                    if traindaynote == '天数无上限':
                        Autonavi_train_valid_df = Autonavi_df[Autonavi_df['Date'] <= date]
                    elif traindaynote == f'天数上限{k}':
                        Autonavi_train_valid_df = Autonavi_df[(Autonavi_df['Date'] >= date - k + 1) & (Autonavi_df['Date'] <= date)]

                    Autonavi_train_df, Autonavi_valid_df = train_test_split(Autonavi_train_valid_df, test_size=0.2, random_state=42, shuffle=True)
                    Autonavi_test_df = Autonavi_df[Autonavi_df['Date'] == test_date].reset_index(drop=True)
                    Autonavi_total_test_df = data_df[data_df['Date'] == test_date].reset_index(drop=True)

                elif testnote == '测试周期抽取':
                    # 获取测试日期范围
                    test_date1, test_date2 = None, None
                    for days in days_set:
                        if date < days[1] + 1:
                            test_date1, test_date2 = days[0], days[1]
                            break
                    if not test_date1 or not test_date2:
                        test_date1, test_date2 = days_set[-1]

                    # 拆分数据
                    Autonavi_df1, Autonavi_df2 = train_test_split(Autonavi_df, test_size=0.2, random_state=42, shuffle=True)
                    if traindaynote == '天数无上限':
                        Autonavi_train_valid_df = Autonavi_df1[Autonavi_df1['Date'] <= date]
                    elif traindaynote == f'天数上限{k}':
                        Autonavi_train_valid_df = Autonavi_df1[(Autonavi_df1['Date'] >= date - k + 1) & (Autonavi_df1['Date'] <= date)]

                    Autonavi_train_df, Autonavi_valid_df = train_test_split(Autonavi_train_valid_df, test_size=0.2, random_state=42, shuffle=True)
                    Autonavi_test_df = Autonavi_df2[(Autonavi_df2['Date'] >= test_date1) & (Autonavi_df2['Date'] <= test_date2)].reset_index(drop=True)
                    Autonavi_total_test_df = data_df[(data_df['Date'] >= test_date1) & (data_df['Date'] <= test_date2)].reset_index(drop=True)

                elif testnote == '测试为下一天':
                    if date == self.date_lst[-1]:
                        continue
                    # 选择训练和验证数据
                    if traindaynote == '天数无上限':
                        Autonavi_train_valid_df = Autonavi_df[Autonavi_df['Date'] <= date]
                    elif traindaynote == f'天数上限{k}':
                        Autonavi_train_valid_df = Autonavi_df[(Autonavi_df['Date'] >= date - k + 1) & (Autonavi_df['Date'] <= date)]

                    Autonavi_train_df, Autonavi_valid_df = train_test_split(Autonavi_train_valid_df, test_size=0.2, random_state=42, shuffle=True)
                    Autonavi_test_df = Autonavi_df[Autonavi_df['Date'] == date + 1].reset_index(drop=True)
                    Autonavi_total_test_df = data_df[data_df['Date'] == date + 1].reset_index(drop=True)

                # 训练模型并进行预测
                Autonavi_model, Autonavi_test_df, Autonavi_total_test_df, Autonavi_performance, Autonavi_total_performance = self.run(
                    Autonavi_train_df, Autonavi_valid_df, Autonavi_test_df, Autonavi_total_test_df, is_search
                )

                # 保存模型
                with open(f'Learning/Model/{note}/n{self.sample_n}_pro{Autonavi_pro}_date{date}_xgb_model.pkl', 'wb') as file:
                    pickle.dump(Autonavi_model, file)

                # 添加性能信息
                Autonavi_performance['Type'] = 'Autonavi'
                Autonavi_performance['Note'] = 'pure'
                Autonavi_performance['Pro'] = Autonavi_pro
                Autonavi_performance['n'] = self.sample_n
                Autonavi_performance['date'] = date

                Autonavi_total_performance['Type'] = 'Autonavi'
                Autonavi_total_performance['Note'] = 'total'
                Autonavi_total_performance['Pro'] = Autonavi_pro
                Autonavi_total_performance['n'] = self.sample_n
                Autonavi_total_performance['date'] = date

                end_time = time.time()

                print(f"pure : MAPE {Autonavi_performance['MAPE']}")
                print(f"total: MAPE {Autonavi_total_performance['MAPE']}")
                print(f"Time : {end_time - start_time} seconds")

                with open(log_path, 'a') as file:
                    file.write(f"pure : MAPE {Autonavi_performance['MAPE']}\n")
                    file.write(f"total: MAPE {Autonavi_total_performance['MAPE']}\n")
                    file.write(f"Time : {end_time - start_time} seconds\n")

                with open(performance_path, 'a') as file:
                    writer = csv.DictWriter(file, fieldnames=fieldnames)
                    writer.writerow(Autonavi_performance)
                    writer.writerow(Autonavi_total_performance)

                # 保存预测结果
                Autonavi_test_df.to_cudf().to_csv(f"Learning/PredictResult/{note}/Autonavi_predictions_n{self.sample_n}_Pro{Autonavi_pro}_date{date}.csv", index=False)

        total_endtime = time.time()
        with open(log_path, 'a') as file:
            file.write(f"---------------------------------------------------------------------------------------------------------\n")
            file.write(f"total use time : {total_endtime - total_starttime} seconds\n")

        # 训练结束后关闭client和cluster
        self.client.close()
        self.cluster.close()

    def drawImportance(self, xgb_model):
        """
        绘制特征重要性
        """
        plt.rcParams['font.sans-serif'] = ['SimHei']
        plot_importance(xgb_model.get_booster())
        plt.show()
