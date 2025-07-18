import os
import csv
import pickle
import math
import time
import pandas as pd
import matplotlib.pyplot as plt
import xgboost as xgb
from xgboost import plot_importance
from sklearn.model_selection import train_test_split,RandomizedSearchCV
from sklearn.metrics import mean_squared_error,mean_absolute_error, r2_score,mean_absolute_percentage_error
import numpy as np


class LearningForRingBySampling:
    def __init__(self,sample_n,n_lst,pro_lst,date_lst,features,target):
        """
        n 是原始抽样文件的滴滴数量
        """
        self.sample_n = sample_n
        self.n_lst = n_lst
        self.pro_lst = pro_lst
        self.date_lst = date_lst
        self.features = features
        self.target = target
    def search_params(self,X_train,y_train,X_val, y_val):
        # 定义 XGBoost 模型
        xgb_model = xgb.XGBRegressor(objective='reg:squarederror',tree_method='hist', enable_categorical=True)

        # 随机搜索参数
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

        # 随机网格搜索
        random_search = RandomizedSearchCV(estimator=xgb_model, param_distributions=param_dist,
                                        n_iter=20, scoring='neg_mean_absolute_percentage_error', cv=5,
                                        verbose=1, n_jobs=-1, random_state=np.random.seed(42))

        # 执行随机搜索
        random_search.fit(X_train, y_train, 
                        eval_set=[(X_val, y_val)],
                        verbose=False)
        
        # 获取最佳参数
        best_params = random_search.best_params_
        print("Best Parameters:", best_params)

        # 获取最佳模型
        xgb_model = random_search.best_estimator_

        return xgb_model

    def getPerformance(self, y_true , y_pred):
        # 计算回归性能指标
        MSE  = mean_squared_error(y_true,y_pred)
        RMSE = math.sqrt(MSE)
        mae  = mean_absolute_error(y_true,y_pred)
        r2  = r2_score(y_true,y_pred)
        mape = mean_absolute_percentage_error(y_true,y_pred)
        return {'MSE':MSE,'RMSE':RMSE,'MAE':mae,'R2':r2,'MAPE':mape}
    
    def drawImportance(self,xgb_model):
        # 画出XGBoost模型的重要性
        plt.rcParams['font.sans-serif'] = ['SimHei']
        # print(xgb_model.feature_importances_)
        plot_importance(xgb_model)
        plt.show()

    def run(self,train_df, valid_df, test_df, total_test_df,is_search):
        if is_search:
            #print("--------------------------------------------------searching parameters--------------------------------------------------")
            xgb_model = self.search_params(train_df[self.features],train_df[self.target],valid_df[self.features],valid_df[self.target])
            ear = xgb_model.get_params()['n_estimators'] * 0.1
            #print("--------------------------------------------------training model--------------------------------------------------")
            xgb_model.set_params(early_stopping_rounds=int(ear),eval_metric='mape')
        else:
            #print("--------------------------------------------------training model--------------------------------------------------")
            xgb_model = xgb.XGBRegressor(
            objective='reg:squarederror',
            tree_method='hist',
            enable_categorical=True,
            n_estimators=1400,
            learning_rate=0.01,
            max_depth=10,
            min_child_weight=2,
            gamma=0.6,
            subsample=0.6,
            colsample_bytree=0.7,
            reg_alpha=3,
            reg_lambda=1,
            early_stopping_rounds=140,
            eval_metric='mape'
        )
        
        xgb_model.fit(train_df[self.features],train_df[self.target],eval_set=[(valid_df[self.features],valid_df[self.target])],verbose=False)
        # drawImportance(xgb_model)

        #print("--------------------------------------------------predictions--------------------------------------------------")
        test_df['Predicted Travel Time'] = xgb_model.predict(test_df[self.features])
        total_test_df['Predicted Travel Time'] = xgb_model.predict(total_test_df[self.features])

        performance = self.getPerformance(test_df[self.target], test_df['Predicted Travel Time'])
        total_performance = self.getPerformance(total_test_df[self.target], total_test_df['Predicted Travel Time'])

        return xgb_model,test_df,total_test_df,performance,total_performance
    

    def learning(self,sub_path,traindaynote,testnote,is_search=False,k=None,days_set=None):
        """
        sub_path:仿真备注，如1209
        traindaynote:训练的天数设置，可选项为 天数无上限/天数上限k
        testnote:测试的策略设置，可选性为 测试周期最后/测试周期抽取/测试为下一天
        k: 只有当traindaynote为天数上限k时才可用
        days_set: 只有当testnote为 测试周期最后 和 测试周期抽取 时才有用。如果一共有30天仿真，红绿灯在第11天和第21天改变，则days_set=[(0,9),(10,19),(20,29)]
        """
        total_starttime = time.time()

        # 将note组合
        note = sub_path+'&'+traindaynote+'&'+testnote+'&sample_n'+str(self.sample_n)+'&is_search_'+str(is_search)

        # 创建路径
        if not os.path.exists(f'Learning/Model/{note}'):
            os.makedirs(f'Learning/Model/{note}')

        if not os.path.exists(f'Learning/PredictResult/{note}'):
            os.makedirs(f'Learning/PredictResult/{note}')

        # 新建日志
        log_path = f"Learning/Logs/logs_{note}.txt"
        with open(log_path, 'w') as file:
            pass

        # 新建performance
        performance_path = f"Learning/Performance/performance_{note}.csv"
        fieldnames = ['MSE','RMSE','MAE','R2','MAPE','Type','Note','Pro','n','date']
        with open(performance_path, 'w') as file:
            writer = csv.DictWriter(file, fieldnames=fieldnames)
            # 写入表头
            writer.writeheader()

        # 可以读取全集数据
        data_df = pd.read_csv(f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{self.sample_n}_total.csv",encoding='utf-8')

        # 开始学习
        for n in self.n_lst:
            # 读取训练数据
            DiDi_df = pd.read_csv(f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{self.sample_n}_num_DiDi{n}.csv",encoding='utf-8')

            for date in self.date_lst:
                print(f"-----------------------------------n : {n} ; Date: {date} ; DiDi Result:-----------------------------------")
                with open(log_path, 'a') as file:
                    file.write(f"-----------------------------------n : {n} ; Date: {date} ; DiDi Result:-----------------------------------\n")

                start_time = time.time()

                ##  针对滴滴
                if testnote == '测试周期最后':
                    # 最后一天不能参与进来学习
                    if date == self.date_lst[-1] : continue
                    # 拿出对应的最后一天
                    for days in days_set:
                        if date < days[1]:
                            test_date = days[1]
                    # 获得 训练 验证 和 测试 数据
                    if traindaynote == '天数无上限':
                        DiDi_train_valid_df = DiDi_df[(DiDi_df['Date']<=date)]
                    elif traindaynote == f'天数上限{k}':
                        DiDi_train_valid_df = DiDi_df[(DiDi_df['Date'].between(date-k+1,date))]
                    
                    DiDi_train_df,DiDi_valid_df = train_test_split(DiDi_train_valid_df, test_size=0.2, random_state=42,shuffle=True)
                    DiDi_test_df = DiDi_df[(DiDi_df['Date']==test_date)].reset_index(drop=True)
                    DiDi_total_test_df = data_df[data_df['Date']==test_date].reset_index(drop=True)

                elif testnote == '测试周期抽取':
                    # 拿出对应的时间范围
                    for days in days_set:
                        if date < days[1]+1:
                            test_date1,test_date2 = days[0],days[1]
                    # 先拿出一部分
                    DiDi_df1,DiDi_df2 = train_test_split(DiDi_df, test_size=0.2, random_state=42,shuffle=True)
                    # 获得 训练 验证 和 测试 数据
                    if traindaynote == '天数无上限':
                        DiDi_train_valid_df = DiDi_df1[(DiDi_df1['Date']<=date)]
                    elif traindaynote == f'天数上限{k}':
                        DiDi_train_valid_df = DiDi_df1[(DiDi_df1['Date'].between(date-k+1,date))]
                    
                    DiDi_train_df,DiDi_valid_df = train_test_split(DiDi_train_valid_df, test_size=0.2, random_state=42,shuffle=True)
                    DiDi_test_df = DiDi_df2[(DiDi_df2['Date'].between(test_date1,test_date2))].reset_index(drop=True)
                    DiDi_total_test_df = data_df[data_df['Date'].between(test_date1,test_date2)].reset_index(drop=True)

                elif testnote == '测试为下一天':
                    # 最后一天不能参与进来学习
                    if date == self.date_lst[-1] : continue
                    # 获得 训练 验证 和 测试 数据
                    if traindaynote == '天数无上限':
                        DiDi_train_valid_df = DiDi_df[(DiDi_df['Date']<=date)]
                    elif traindaynote == f'天数上限{k}':
                        DiDi_train_valid_df = DiDi_df[(DiDi_df['Date'].between(date-k+1,date))]
                    
                    DiDi_train_df,DiDi_valid_df = train_test_split(DiDi_train_valid_df, test_size=0.2, random_state=42, shuffle=True)
                    DiDi_test_df = DiDi_df[(DiDi_df['Date']==date+1)].reset_index(drop=True)
                    DiDi_total_test_df = data_df[data_df['Date']==date+1].reset_index(drop=True)

                # 进行学习
                DiDi_model,DiDi_test_df,DiDi_total_test_df,DiDi_performance,DiDi_total_performance = self.run(DiDi_train_df, DiDi_valid_df, DiDi_test_df, DiDi_total_test_df,is_search)

                with open(f'Learning/Model/{note}/n{self.sample_n}_num_DiDi{n}_date{date}_xgb_model.pkl', 'wb') as file:
                    pickle.dump(DiDi_model, file)


                DiDi_performance['Type'],DiDi_performance['Note'],DiDi_performance['Pro'],DiDi_performance['n'],DiDi_performance['date'] = 'DiDi','pure',DiDi_df.shape[0]/data_df.shape[0],n,date
                DiDi_total_performance['Type'],DiDi_total_performance['Note'],DiDi_total_performance['Pro'],DiDi_total_performance['n'],DiDi_total_performance['date'] = 'DiDi','total',DiDi_df.shape[0]/data_df.shape[0],n,date
                
                end_time = time.time()
                

                print('pure : MAPE '+str(DiDi_performance['MAPE']))
                print('total: MAPE '+str(DiDi_total_performance['MAPE']))
                print(f"Time : {end_time - start_time} seconds")
                with open(log_path, 'a') as file:
                    file.write('pure : MAPE '+str(DiDi_performance['MAPE'])+'\n')
                    file.write('total: MAPE '+str(DiDi_total_performance['MAPE'])+'\n')
                    file.write(f"Time : {end_time - start_time} seconds"+'\n')

                with open(performance_path, 'a') as file:
                    writer = csv.DictWriter(file, fieldnames=fieldnames)
                    # 逐行写入数据
                    writer.writerow(DiDi_performance)
                    writer.writerow(DiDi_total_performance)

                DiDi_test_df.to_csv(f"Learning/PredictResult/{note}/DiDi_predictions_n{self.sample_n}_num_DiDi{n}_date{date}.csv",index=None)


        # 高德数据
        for Autonavi_pro in self.pro_lst:
            Autonavi_pro = round(Autonavi_pro,3)

            for date in self.date_lst:
                print(f"-----------------------------------Date: {date} ; Autonavi {Autonavi_pro} Result:-----------------------------------")
                with open(log_path, 'a') as file:
                    file.write(f"-----------------------------------Date: {date} ; Autonavi {Autonavi_pro} Result:-----------------------------------\n")

                start_time = time.time()

                Autonavi_df = pd.read_csv(f"Learning/TimeFeatureBySampling/{sub_path}/VehicleData_n{self.sample_n}_pro{Autonavi_pro}.csv",encoding='utf-8')
                if testnote == '测试周期最后':
                    # 最后一天不能参与进来学习
                    if date == self.date_lst[-1] : continue
                    # 拿出对应的最后一天
                    for days in days_set:
                        if date < days[1]:
                            test_date = days[1]
                    # 获得 训练 验证 和 测试 数据
                    if traindaynote == '天数无上限':
                        Autonavi_train_valid_df = Autonavi_df[(Autonavi_df['Date']<=date)]
                    elif traindaynote == f'天数上限{k}':
                        Autonavi_train_valid_df = Autonavi_df[(Autonavi_df['Date'].between(date-k+1,date))]
                    
                    Autonavi_train_df,Autonavi_valid_df = train_test_split(Autonavi_train_valid_df, test_size=0.2, random_state=42,shuffle=True)
                    Autonavi_test_df = Autonavi_df[(Autonavi_df['Date']==test_date)].reset_index(drop=True)
                    Autonavi_total_test_df = data_df[data_df['Date']==test_date].reset_index(drop=True)

                elif testnote == '测试周期抽取':
                    # 拿出对应的时间范围
                    for days in days_set:
                        if date < days[1]+1:
                            test_date1,test_date2 = days[0],days[1]
                    # 先拿出一部分
                    Autonavi_df1,Autonavi_df2 = train_test_split(Autonavi_df, test_size=0.2, random_state=42,shuffle=True)
                    # 获得 训练 验证 和 测试 数据
                    if traindaynote == '天数无上限':
                        Autonavi_train_valid_df = Autonavi_df1[(Autonavi_df1['Date']<=date)]
                    elif traindaynote == f'天数上限{k}':
                        Autonavi_train_valid_df = Autonavi_df1[(Autonavi_df1['Date'].between(date-k+1,date))]
                    
                    Autonavi_train_df,Autonavi_valid_df = train_test_split(Autonavi_train_valid_df, test_size=0.2, random_state=42,shuffle=True)
                    Autonavi_test_df = Autonavi_df2[(Autonavi_df2['Date'].between(test_date1,test_date2))].reset_index(drop=True)
                    Autonavi_total_test_df = data_df[data_df['Date'].between(test_date1,test_date2)].reset_index(drop=True)

                elif testnote == '测试为下一天':
                    # 最后一天不能参与进来学习
                    if date == self.date_lst[-1] : continue
                    # 获得 训练 验证 和 测试 数据
                    if traindaynote == '天数无上限':
                        Autonavi_train_valid_df = Autonavi_df[(Autonavi_df['Date']<=date)]
                    elif traindaynote == f'天数上限{k}':
                        Autonavi_train_valid_df = Autonavi_df[(Autonavi_df['Date'].between(date-k+1,date))]
                    
                    Autonavi_train_df,Autonavi_valid_df = train_test_split(Autonavi_train_valid_df, test_size=0.2, random_state=42, shuffle=True)
                    Autonavi_test_df = Autonavi_df[(Autonavi_df['Date']==date+1)].reset_index(drop=True)
                    Autonavi_total_test_df = data_df[data_df['Date']==date+1].reset_index(drop=True)

                Autonavi_model,Autonavi_test_df,Autonavi_total_test_df,Autonavi_performance,Autonavi_total_performance = self.run(Autonavi_train_df,Autonavi_valid_df,Autonavi_test_df,Autonavi_total_test_df,is_search)

                with open(f'Learning/Model/{note}/n{self.sample_n}_pro{Autonavi_pro}_date{date}_xgb_model.pkl', 'wb') as file:
                    pickle.dump(Autonavi_model, file)

                Autonavi_performance['Type'],Autonavi_performance['Note'],Autonavi_performance['Pro'],Autonavi_performance['n'],Autonavi_performance['date'] = 'Autonavi','pure',Autonavi_pro,n,date
                Autonavi_total_performance['Type'],Autonavi_total_performance['Note'],Autonavi_total_performance['Pro'],Autonavi_total_performance['n'],Autonavi_total_performance['date'] = 'Autonavi','total',Autonavi_pro,n,date

                end_time = time.time()
                

                print('pure : MAPE '+str(Autonavi_performance['MAPE']))
                print('total: MAPE '+str(Autonavi_total_performance['MAPE']))
                print(f"Time : {end_time - start_time} seconds")
                with open(log_path, 'a') as file:
                    file.write('pure : MAPE '+str(Autonavi_performance['MAPE'])+'\n')
                    file.write('total: MAPE '+str(Autonavi_total_performance['MAPE'])+'\n')
                    file.write(f"Time : {end_time - start_time} seconds"+'\n')

                with open(performance_path, 'a') as file:
                    writer = csv.DictWriter(file, fieldnames=fieldnames)
                    # 逐行写入数据
                    writer.writerow(Autonavi_performance)
                    writer.writerow(Autonavi_total_performance)


                Autonavi_test_df.to_csv(f"Learning/PredictResult/{note}/Autonavi_predictions_n{self.sample_n}_Pro{Autonavi_pro}_date{date}.csv",index=None)
    
        total_endtime = time.time()
        with open(log_path, 'a') as file:
            file.write(f"---------------------------------------------------------------------------------------------------------\n")
            file.write(f"total use time : {total_endtime - total_starttime} seconds\n")